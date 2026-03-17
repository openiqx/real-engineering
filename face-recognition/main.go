package main

import (
	"fmt"
	"math/rand"
)


// Photo represents an uploaded photo containing one or more faces.
type Photo struct {
	ID    string
	Faces []Face
}

// Face represents a single detected face in a photo.
type Face struct {
	ID        string
	PhotoID   string
	Embedding Embedding
}

// pipeline ties all stages together
// detect faces -> generate embeddings -> cluster -> check for merges.
func pipeline(photo Photo, store *ClusterStore) {
	for _, face := range photo.Faces {
		// Find the closest existing cluster for this embedding.
		// If confidence is high enough, merge into it.
		// If not, create a new unnamed cluster and wait.
		clusterID := store.Assign(face)

		fmt.Printf("face %s assigned to cluster %s\n", face.ID, clusterID)
	}

	// After every new photo, check whether any two clusters
	// have become connected through a chain of intermediate embeddings.
	merged := store.CheckAndMerge()
	for _, m := range merged {
		fmt.Printf("clusters merged: %s + %s -> %s\n", m.A, m.B, m.Result)
	}
}

func main() {
	store := NewClusterStore(0.6, 0.75)

	// Simulate the same person across 10 years of photos.
	// Each embedding drifts slightly from the previous one.
	// no single step is too large, but the total drift is important.
	// Age 8 and age 40 are far apart.
	// But the chain connects them through intermediate photos.
 
	photos := []Photo{
		{
			ID: "photo_age_08",
			Faces: []Face{
				{ID: "face_001", PhotoID: "photo_age_08", Embedding: randomEmbedding(0.10, 0.05)},
			},
		},
		{
			ID: "photo_age_12",
			Faces: []Face{
				{ID: "face_002", PhotoID: "photo_age_12", Embedding: randomEmbedding(0.22, 0.05)},
			},
		},
		{
			ID: "photo_age_18",
			Faces: []Face{
				{ID: "face_003", PhotoID: "photo_age_18", Embedding: randomEmbedding(0.38, 0.05)},
			},
		},
		{
			ID: "photo_age_25",
			Faces: []Face{
				{ID: "face_004", PhotoID: "photo_age_25", Embedding: randomEmbedding(0.55, 0.05)},
			},
		},
		{
			ID: "photo_age_35",
			Faces: []Face{
				{ID: "face_005", PhotoID: "photo_age_35", Embedding: randomEmbedding(0.68, 0.05)},
			},
		},
		{
			ID: "photo_age_40",
			Faces: []Face{
				{ID: "face_006", PhotoID: "photo_age_40", Embedding: randomEmbedding(0.80, 0.05)},
			},
		},
	}
 
	fmt.Println("-- processing photos --")
	for _, photo := range photos {
		pipeline(photo, store)
	}
 
	fmt.Println("\n-- final clusters --")
	for id, cluster := range store.Clusters {
		fmt.Printf("cluster %s: %d faces\n", id, len(cluster.FaceIDs))
	}
}

// randomEmbedding generates a simple 1D embedding near center value.
func randomEmbedding(center, jitter float64) Embedding {
	return Embedding{
		Values: []float64{
			center + (rand.Float64()*2-1)*jitter,
		},
	}
}