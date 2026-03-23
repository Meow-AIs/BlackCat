package memory

import (
	"context"
	"math"
	"testing"
)

func TestNewSimpleEmbedder(t *testing.T) {
	e := NewSimpleEmbedder(384)
	if e == nil {
		t.Fatal("NewSimpleEmbedder returned nil")
	}
	if e.Dimensions() != 384 {
		t.Errorf("Dimensions = %d, want 384", e.Dimensions())
	}
}

func TestSimpleEmbedder_Embed(t *testing.T) {
	e := NewSimpleEmbedder(384)
	ctx := context.Background()

	vec, err := e.Embed(ctx, "Hello world")
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	if len(vec) != 384 {
		t.Fatalf("vector length = %d, want 384", len(vec))
	}

	// Values should be bounded [-1, 1]
	for i, v := range vec {
		if v < -1.0 || v > 1.0 {
			t.Errorf("vec[%d] = %f, out of [-1, 1]", i, v)
			break
		}
	}
}

func TestSimpleEmbedder_Deterministic(t *testing.T) {
	e := NewSimpleEmbedder(384)
	ctx := context.Background()

	v1, _ := e.Embed(ctx, "Same text")
	v2, _ := e.Embed(ctx, "Same text")

	for i := range v1 {
		if v1[i] != v2[i] {
			t.Errorf("embeddings differ at index %d: %f vs %f", i, v1[i], v2[i])
			break
		}
	}
}

func TestSimpleEmbedder_DifferentTexts(t *testing.T) {
	e := NewSimpleEmbedder(384)
	ctx := context.Background()

	v1, _ := e.Embed(ctx, "Go programming language")
	v2, _ := e.Embed(ctx, "Python programming language")

	// Should produce different vectors
	same := true
	for i := range v1 {
		if v1[i] != v2[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("different texts should produce different vectors")
	}
}

func TestSimpleEmbedder_EmbedBatch(t *testing.T) {
	e := NewSimpleEmbedder(128)
	ctx := context.Background()

	texts := []string{"Hello", "World", "Foo"}
	vecs, err := e.EmbedBatch(ctx, texts)
	if err != nil {
		t.Fatalf("EmbedBatch failed: %v", err)
	}

	if len(vecs) != 3 {
		t.Fatalf("got %d vectors, want 3", len(vecs))
	}

	for i, v := range vecs {
		if len(v) != 128 {
			t.Errorf("vec[%d] length = %d, want 128", i, len(v))
		}
	}

	// Each should match individual embed
	for i, text := range texts {
		single, _ := e.Embed(ctx, text)
		for j := range single {
			if single[j] != vecs[i][j] {
				t.Errorf("batch[%d] differs from single at index %d", i, j)
				break
			}
		}
	}
}

func TestSimpleEmbedder_EmbedBatch_Empty(t *testing.T) {
	e := NewSimpleEmbedder(128)
	ctx := context.Background()

	vecs, err := e.EmbedBatch(ctx, nil)
	if err != nil {
		t.Fatalf("EmbedBatch failed: %v", err)
	}
	if len(vecs) != 0 {
		t.Errorf("got %d vectors, want 0", len(vecs))
	}
}

func TestSimpleEmbedder_Normalized(t *testing.T) {
	e := NewSimpleEmbedder(128)
	ctx := context.Background()

	vec, _ := e.Embed(ctx, "Test normalization")

	// Compute L2 norm
	var sum float64
	for _, v := range vec {
		sum += float64(v) * float64(v)
	}
	norm := math.Sqrt(sum)

	// Should be approximately 1.0 (unit vector)
	if math.Abs(norm-1.0) > 0.01 {
		t.Errorf("L2 norm = %f, want ~1.0", norm)
	}
}
