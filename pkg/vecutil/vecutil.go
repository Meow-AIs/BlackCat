// Package vecutil provides vector math utilities for embedding operations.
package vecutil

import "math"

// CosineSimilarity computes the cosine similarity between two vectors.
// Returns a value between -1 and 1. Returns 0 if either vector has zero magnitude.
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	dot := DotProduct(a, b)
	magA := magnitude(a)
	magB := magnitude(b)

	if magA == 0 || magB == 0 {
		return 0
	}
	return dot / (magA * magB)
}

// Normalize returns a unit-length copy of the vector.
// Returns a zero-length slice if the input is empty or has zero magnitude.
func Normalize(v []float32) []float32 {
	if len(v) == 0 {
		return []float32{}
	}

	mag := magnitude(v)
	if mag == 0 {
		result := make([]float32, len(v))
		return result
	}

	result := make([]float32, len(v))
	magF := float32(mag)
	for i, val := range v {
		result[i] = val / magF
	}
	return result
}

// QuantizeInt8 converts float32 vector to int8 by scaling to [-127, 127].
// Uses the maximum absolute value as the scale factor.
func QuantizeInt8(v []float32) []int8 {
	if len(v) == 0 {
		return []int8{}
	}

	maxAbs := float32(0)
	for _, val := range v {
		abs := val
		if abs < 0 {
			abs = -abs
		}
		if abs > maxAbs {
			maxAbs = abs
		}
	}

	result := make([]int8, len(v))
	if maxAbs == 0 {
		return result
	}

	scale := float32(127) / maxAbs
	for i, val := range v {
		scaled := val * scale
		if scaled > 127 {
			scaled = 127
		} else if scaled < -127 {
			scaled = -127
		}
		result[i] = int8(scaled)
	}
	return result
}

// DequantizeInt8 converts int8 vector back to float32.
// Values are scaled to [-1, 1] range.
func DequantizeInt8(v []int8) []float32 {
	if len(v) == 0 {
		return []float32{}
	}

	result := make([]float32, len(v))
	for i, val := range v {
		result[i] = float32(val) / 127.0
	}
	return result
}

// DotProduct computes the dot product of two float32 vectors.
// Returns 0 if vectors have different lengths or are empty.
func DotProduct(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var sum float64
	for i := range a {
		sum += float64(a[i]) * float64(b[i])
	}
	return sum
}

// magnitude returns the L2 norm (Euclidean length) of a vector.
func magnitude(v []float32) float64 {
	var sum float64
	for _, val := range v {
		sum += float64(val) * float64(val)
	}
	return math.Sqrt(sum)
}
