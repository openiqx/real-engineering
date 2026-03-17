package main

import "math"

// Embedding is a list of numbers representing the geometry of a face.
// We are using a simplified 1D version so the distance math stays readable
type Embedding struct {
	Values []float64
}

// Distance calculates the Euclidean distance between two embeddings.
//
// Small distance = similar faces => likely same person
// Large distance = different faces => likely different people.
func (a Embedding) Distance(b Embedding) float64 {
	if len(a.Values) != len(b.Values) {
		return math.MaxFloat64
	}

	sum := 0.0
	for i := range a.Values {
		diff := a.Values[i] - b.Values[i]
		sum += diff * diff
	}

	return math.Sqrt(sum)
}
