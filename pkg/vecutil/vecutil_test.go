package vecutil

import (
	"math"
	"testing"
)

const epsilon = 1e-6

func approxEqual(a, b, eps float64) bool {
	return math.Abs(a-b) < eps
}

func TestCosineSimilarityIdentical(t *testing.T) {
	a := []float32{1, 2, 3}
	sim := CosineSimilarity(a, a)
	if !approxEqual(sim, 1.0, epsilon) {
		t.Errorf("identical vectors should have similarity 1.0, got %f", sim)
	}
}

func TestCosineSimilarityOpposite(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{-1, 0, 0}
	sim := CosineSimilarity(a, b)
	if !approxEqual(sim, -1.0, epsilon) {
		t.Errorf("opposite vectors should have similarity -1.0, got %f", sim)
	}
}

func TestCosineSimilarityOrthogonal(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{0, 1, 0}
	sim := CosineSimilarity(a, b)
	if !approxEqual(sim, 0.0, epsilon) {
		t.Errorf("orthogonal vectors should have similarity 0.0, got %f", sim)
	}
}

func TestCosineSimilarityDifferentLengths(t *testing.T) {
	a := []float32{1, 2}
	b := []float32{1, 2, 3}
	sim := CosineSimilarity(a, b)
	if sim != 0 {
		t.Errorf("different length vectors should return 0, got %f", sim)
	}
}

func TestCosineSimilarityEmpty(t *testing.T) {
	sim := CosineSimilarity([]float32{}, []float32{})
	if sim != 0 {
		t.Errorf("empty vectors should return 0, got %f", sim)
	}
}

func TestCosineSimilarityZeroVector(t *testing.T) {
	a := []float32{0, 0, 0}
	b := []float32{1, 2, 3}
	sim := CosineSimilarity(a, b)
	if sim != 0 {
		t.Errorf("zero magnitude vector should return 0, got %f", sim)
	}
}

func TestNormalize(t *testing.T) {
	v := []float32{3, 4}
	result := Normalize(v)
	if len(result) != 2 {
		t.Fatalf("expected length 2, got %d", len(result))
	}
	if !approxEqual(float64(result[0]), 0.6, epsilon) {
		t.Errorf("expected 0.6, got %f", result[0])
	}
	if !approxEqual(float64(result[1]), 0.8, epsilon) {
		t.Errorf("expected 0.8, got %f", result[1])
	}
}

func TestNormalizeUnitLength(t *testing.T) {
	v := []float32{1, 2, 3, 4, 5}
	result := Normalize(v)
	mag := magnitude(result)
	if !approxEqual(mag, 1.0, epsilon) {
		t.Errorf("normalized vector should have magnitude 1.0, got %f", mag)
	}
}

func TestNormalizeEmpty(t *testing.T) {
	result := Normalize([]float32{})
	if len(result) != 0 {
		t.Error("normalizing empty vector should return empty")
	}
}

func TestNormalizeZeroVector(t *testing.T) {
	result := Normalize([]float32{0, 0, 0})
	if len(result) != 3 {
		t.Fatalf("expected length 3, got %d", len(result))
	}
	for i, v := range result {
		if v != 0 {
			t.Errorf("index %d: expected 0, got %f", i, v)
		}
	}
}

func TestNormalizeImmutability(t *testing.T) {
	original := []float32{3, 4}
	_ = Normalize(original)
	if original[0] != 3 || original[1] != 4 {
		t.Error("Normalize should not mutate the input")
	}
}

func TestQuantizeInt8Basic(t *testing.T) {
	v := []float32{1.0, -1.0, 0.5, 0.0}
	result := QuantizeInt8(v)
	if len(result) != 4 {
		t.Fatalf("expected length 4, got %d", len(result))
	}
	if result[0] != 127 {
		t.Errorf("expected 127 for max value, got %d", result[0])
	}
	if result[1] != -127 {
		t.Errorf("expected -127 for min value, got %d", result[1])
	}
	if result[3] != 0 {
		t.Errorf("expected 0 for zero value, got %d", result[3])
	}
}

func TestQuantizeInt8Empty(t *testing.T) {
	result := QuantizeInt8([]float32{})
	if len(result) != 0 {
		t.Error("quantizing empty vector should return empty")
	}
}

func TestQuantizeInt8ZeroVector(t *testing.T) {
	result := QuantizeInt8([]float32{0, 0, 0})
	for i, v := range result {
		if v != 0 {
			t.Errorf("index %d: expected 0, got %d", i, v)
		}
	}
}

func TestQuantizeInt8Immutability(t *testing.T) {
	original := []float32{1.0, -0.5}
	_ = QuantizeInt8(original)
	if original[0] != 1.0 || original[1] != -0.5 {
		t.Error("QuantizeInt8 should not mutate the input")
	}
}

func TestDequantizeInt8Basic(t *testing.T) {
	v := []int8{127, -127, 0, 63}
	result := DequantizeInt8(v)
	if len(result) != 4 {
		t.Fatalf("expected length 4, got %d", len(result))
	}
	if !approxEqual(float64(result[0]), 1.0, 0.01) {
		t.Errorf("expected ~1.0, got %f", result[0])
	}
	if !approxEqual(float64(result[1]), -1.0, 0.01) {
		t.Errorf("expected ~-1.0, got %f", result[1])
	}
	if result[2] != 0 {
		t.Errorf("expected 0, got %f", result[2])
	}
}

func TestDequantizeInt8Empty(t *testing.T) {
	result := DequantizeInt8([]int8{})
	if len(result) != 0 {
		t.Error("dequantizing empty vector should return empty")
	}
}

func TestQuantizeDequantizeRoundTrip(t *testing.T) {
	original := []float32{0.5, -0.3, 0.8, -1.0, 0.0}
	quantized := QuantizeInt8(original)
	recovered := DequantizeInt8(quantized)

	for i := range original {
		if !approxEqual(float64(original[i]), float64(recovered[i]), 0.02) {
			t.Errorf("index %d: original %f, recovered %f", i, original[i], recovered[i])
		}
	}
}

func TestDotProduct(t *testing.T) {
	a := []float32{1, 2, 3}
	b := []float32{4, 5, 6}
	// 1*4 + 2*5 + 3*6 = 4 + 10 + 18 = 32
	result := DotProduct(a, b)
	if !approxEqual(result, 32.0, epsilon) {
		t.Errorf("expected 32.0, got %f", result)
	}
}

func TestDotProductDifferentLengths(t *testing.T) {
	a := []float32{1, 2}
	b := []float32{1, 2, 3}
	result := DotProduct(a, b)
	if result != 0 {
		t.Errorf("different length vectors should return 0, got %f", result)
	}
}

func TestDotProductEmpty(t *testing.T) {
	result := DotProduct([]float32{}, []float32{})
	if result != 0 {
		t.Errorf("empty vectors should return 0, got %f", result)
	}
}

func TestDotProductOrthogonal(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{0, 1, 0}
	result := DotProduct(a, b)
	if !approxEqual(result, 0.0, epsilon) {
		t.Errorf("orthogonal vectors should have dot product 0, got %f", result)
	}
}

func BenchmarkCosineSimilarity384(b *testing.B) {
	v1 := make([]float32, 384)
	v2 := make([]float32, 384)
	for i := range v1 {
		v1[i] = float32(i) * 0.01
		v2[i] = float32(i) * 0.02
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CosineSimilarity(v1, v2)
	}
}

func BenchmarkQuantizeInt8_384(b *testing.B) {
	v := make([]float32, 384)
	for i := range v {
		v[i] = float32(i) * 0.01
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		QuantizeInt8(v)
	}
}

func BenchmarkNormalize384(b *testing.B) {
	v := make([]float32, 384)
	for i := range v {
		v[i] = float32(i) * 0.01
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Normalize(v)
	}
}
