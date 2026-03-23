package memory

import (
	"context"
	"hash/fnv"
	"math"
)

// SimpleEmbedder generates deterministic pseudo-embeddings using hashing.
// It is intended for testing and development; the real ONNX-based embedder
// will replace it later. This implementation does not require CGo.
type SimpleEmbedder struct {
	dimensions int
}

// NewSimpleEmbedder creates an embedder that produces vectors of the given
// dimensionality.
func NewSimpleEmbedder(dims int) *SimpleEmbedder {
	return &SimpleEmbedder{dimensions: dims}
}

// Embed returns a normalized pseudo-embedding for the given text.
// The same text always produces the same vector (deterministic).
func (e *SimpleEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	vec := e.hash(text)
	return normalize(vec), nil
}

// EmbedBatch returns embeddings for multiple texts. Each embedding is
// identical to calling Embed individually.
func (e *SimpleEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i, text := range texts {
		vec, err := e.Embed(ctx, text)
		if err != nil {
			return nil, err
		}
		result[i] = vec
	}
	return result, nil
}

// Dimensions returns the embedding vector size.
func (e *SimpleEmbedder) Dimensions() int {
	return e.dimensions
}

// hash generates a deterministic float32 vector from text using FNV hashing.
// Each dimension is seeded by hashing the text with a dimension-specific salt.
func (e *SimpleEmbedder) hash(text string) []float32 {
	vec := make([]float32, e.dimensions)
	for i := range vec {
		h := fnv.New64a()
		// Salt each dimension with its index
		salt := []byte{byte(i), byte(i >> 8), byte(i >> 16), byte(i >> 24)}
		h.Write(salt)
		h.Write([]byte(text))

		// Map hash to [-1, 1] range
		bits := h.Sum64()
		vec[i] = float32(bits)/float32(math.MaxUint64)*2 - 1
	}
	return vec
}

// normalize scales a vector to unit length (L2 norm = 1).
func normalize(vec []float32) []float32 {
	var sum float64
	for _, v := range vec {
		sum += float64(v) * float64(v)
	}
	norm := math.Sqrt(sum)
	if norm == 0 {
		return vec
	}

	result := make([]float32, len(vec))
	for i, v := range vec {
		result[i] = float32(float64(v) / norm)
	}
	return result
}
