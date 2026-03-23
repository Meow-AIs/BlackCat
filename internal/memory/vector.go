//go:build cgo

package memory

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
)

// VectorMatch holds a memory ID and its similarity score from a KNN search.
type VectorMatch struct {
	MemoryID   string  `json:"memory_id"`
	Similarity float64 `json:"similarity"`
}

// VectorStore manages vector embeddings stored as BLOBs in SQLite.
// For production use with sqlite-vec, the KNN search would use the vec0
// virtual table. This implementation uses a brute-force scan for correctness
// until the sqlite-vec extension is wired in.
type VectorStore struct {
	db *sql.DB
}

// NewVectorStore creates a vector store backed by the given database.
func NewVectorStore(db *sql.DB) *VectorStore {
	return &VectorStore{db: db}
}

// Insert stores a vector embedding for the given memory ID.
func (vs *VectorStore) Insert(memoryID string, embedding []float32) error {
	blob := encodeFloat32Slice(embedding)

	_, err := vs.db.Exec(
		`INSERT OR REPLACE INTO memory_vectors (memory_id, embedding) VALUES (?, ?)`,
		memoryID, blob,
	)
	if err != nil {
		return fmt.Errorf("insert vector: %w", err)
	}
	return nil
}

// SearchKNN performs a brute-force K-nearest-neighbor search using cosine
// similarity. Returns up to limit results ordered by descending similarity.
func (vs *VectorStore) SearchKNN(query []float32, limit int) ([]VectorMatch, error) {
	rows, err := vs.db.Query(`SELECT memory_id, embedding FROM memory_vectors`)
	if err != nil {
		return nil, fmt.Errorf("query vectors: %w", err)
	}
	defer rows.Close()

	type scored struct {
		memoryID   string
		similarity float64
	}

	var candidates []scored
	for rows.Next() {
		var memoryID string
		var blob []byte
		if err := rows.Scan(&memoryID, &blob); err != nil {
			continue
		}

		vec := decodeFloat32Slice(blob)
		sim := cosineSimilarity(query, vec)
		candidates = append(candidates, scored{memoryID: memoryID, similarity: sim})
	}

	// Sort by similarity descending (insertion sort for small N)
	for i := 1; i < len(candidates); i++ {
		for j := i; j > 0 && candidates[j].similarity > candidates[j-1].similarity; j-- {
			candidates[j], candidates[j-1] = candidates[j-1], candidates[j]
		}
	}

	if limit > len(candidates) {
		limit = len(candidates)
	}

	results := make([]VectorMatch, limit)
	for i := 0; i < limit; i++ {
		results[i] = VectorMatch{
			MemoryID:   candidates[i].memoryID,
			Similarity: candidates[i].similarity,
		}
	}
	return results, nil
}

// Delete removes the vector for the given memory ID.
func (vs *VectorStore) Delete(memoryID string) error {
	_, err := vs.db.Exec(`DELETE FROM memory_vectors WHERE memory_id = ?`, memoryID)
	if err != nil {
		return fmt.Errorf("delete vector: %w", err)
	}
	return nil
}

// encodeFloat32Slice converts a float32 slice to a little-endian byte slice.
func encodeFloat32Slice(v []float32) []byte {
	buf := make([]byte, len(v)*4)
	for i, f := range v {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

// decodeFloat32Slice converts a little-endian byte slice back to float32s.
func decodeFloat32Slice(b []byte) []float32 {
	n := len(b) / 4
	v := make([]float32, n)
	for i := 0; i < n; i++ {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return v
}

// cosineSimilarity computes the cosine similarity between two vectors.
// Returns 0 if either vector has zero magnitude.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return dot / denom
}
