package memory

import "context"

// Tier identifies which memory tier an entry belongs to.
type Tier string

const (
	TierEpisodic   Tier = "episodic"
	TierSemantic   Tier = "semantic"
	TierProcedural Tier = "procedural"
)

// Entry is a single memory record.
type Entry struct {
	ID        string            `json:"id"`
	Tier      Tier              `json:"tier"`
	Content   string            `json:"content"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Score     float64           `json:"score"`      // relevance/decay score
	CreatedAt int64             `json:"created_at"`  // unix timestamp
	UpdatedAt int64             `json:"updated_at"`
	UserID    string            `json:"user_id,omitempty"` // for channel scoping
}

// SearchQuery describes a memory search request.
type SearchQuery struct {
	Text     string            `json:"text"`
	Tier     Tier              `json:"tier,omitempty"`    // filter by tier
	UserID   string            `json:"user_id,omitempty"` // filter by user
	Metadata map[string]string `json:"metadata,omitempty"`
	Limit    int               `json:"limit"`
}

// SearchResult is a memory entry with a relevance score.
type SearchResult struct {
	Entry     Entry   `json:"entry"`
	Relevance float64 `json:"relevance"` // 0.0 - 1.0
}

// Snapshot is a frozen memory context injected at session start.
type Snapshot struct {
	Content    string `json:"content"`     // compiled text (~800 tokens)
	TokenCount int    `json:"token_count"`
	EntryCount int    `json:"entry_count"`
}

// Stats contains memory usage statistics.
type Stats struct {
	TotalEntries    int   `json:"total_entries"`
	EpisodicCount   int   `json:"episodic_count"`
	SemanticCount   int   `json:"semantic_count"`
	ProceduralCount int   `json:"procedural_count"`
	VectorCount     int   `json:"vector_count"`
	DBSizeBytes     int64 `json:"db_size_bytes"`
}

// Engine is the interface for the memory subsystem.
type Engine interface {
	// Store saves a memory entry and its vector embedding.
	Store(ctx context.Context, entry Entry) error

	// Search performs hybrid retrieval (FTS5 + vector + metadata).
	Search(ctx context.Context, query SearchQuery) ([]SearchResult, error)

	// Delete removes a memory entry by ID.
	Delete(ctx context.Context, id string) error

	// BuildSnapshot creates a frozen context snapshot for session start.
	BuildSnapshot(ctx context.Context, projectID string, userID string) (Snapshot, error)

	// Decay runs time-weighted decay and evicts entries over budget.
	Decay(ctx context.Context) (int, error)

	// Stats returns memory usage statistics.
	Stats(ctx context.Context) (Stats, error)

	// Close cleanly shuts down the memory engine.
	Close() error
}

// Embedder generates vector embeddings from text.
type Embedder interface {
	// Embed returns a vector embedding for the given text.
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch returns embeddings for multiple texts.
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

	// Dimensions returns the embedding vector size.
	Dimensions() int
}
