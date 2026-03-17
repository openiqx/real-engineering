package main

import (
	"fmt"
	"sync"
)

// Cluster is a group of face embeddings the system believes
// belong to the same person. It has no name until the user assigns one.
//
// The centroid is the average embedding of all faces in the cluster
// it drifts over time as new faces are added, which is exactly what
// allows the system to track a face across years of aging.
type Cluster struct {
	ID       string
	FaceIDs  []string
	Centroid Embedding
}

// MergeEvent record when two clusters were combined.
type MergeEvent struct {
	A      string
	B      string
	Result string
}

// ClusterStore holds all clusters and the logic for assigning
// new faces and merging connected clusters.
type ClusterStore struct {
	mu       sync.Mutex
	Clusters map[string]*Cluster

	// confidenceThreshold: min similarity to merge a face
	// into an existing cluster immediately.
	confidenceThreshold float64

	// mergeThreshold: minimum similarity between two cluster
	// centroids to consider merging them via chain connectivity.
	mergeThreshold float64

	nextID int
}

func NewClusterStore(confidenceThreshold, mergeThreshold float64) *ClusterStore {
	return &ClusterStore {
		Clusters: make(map[string]*Cluster),
		confidenceThreshold: confidenceThreshold,
		mergeThreshold: mergeThreshold,
	}
}

// Assign finds the best existing cluster for a face embedding.
//
// If a cluster is close enough i.e. within confidenceThreshold, the face
// is merged in and the centroid is updated.
//
// If no cluster is close enough, a new unnamed cluster is created and 
// the system waits for more evidence before deciding.
func (cs *ClusterStore) Assign(face Face) string {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	bestClusterID := ""
	bestDistance := cs.confidenceThreshold

	for id, cluster := range cs.Clusters {
		dist := face.Embedding.Distance(cluster.Centroid)
		if dist < bestDistance {
			bestDistance = dist
			bestClusterID = id
		}
	}

	if bestClusterID != "" {
		cs.Clusters[bestClusterID].FaceIDs = append(cs.Clusters[bestClusterID].FaceIDs, face.ID)
		cs.Clusters[bestClusterID].Centroid = cs.updateCentroid(
			cs.Clusters[bestClusterID].Centroid,
			face.Embedding,
			len(cs.Clusters[bestClusterID].FaceIDs),
		)
		return bestClusterID
	}

	// No confident match, create a new unnamed cluster.
	// It will be merged later if subsequent photos bring it closer
	// to an existing cluster.
	newID := cs.newClusterID()
	cs.Clusters[newID] = &Cluster{
		ID:       newID,
		FaceIDs:  []string{face.ID},
		Centroid: face.Embedding,
	}
	return newID
}

// CheckAndMerge looks for pairs of clusters whose centroids are close
// enough to suggest they belong to the same person
//
// This is the chain connectivity check from the architectural flow diagram. It runs
// after every new photo is processed. As new photos arrive and centroids drift closer
// together, cluster that we previously to far apart eventually become mergeable
func (cs *ClusterStore) CheckAndMerge() []MergeEvent {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	var events []MergeEvent

	ids := make([]string, 0, len(cs.Clusters))
	for id := range cs.Clusters {
		ids = append(ids, id)
	}

	merged := make(map[string]bool)

	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			a := ids[i]
			b := ids[j]

			if merged[a] || merged[b] {
				continue
			}

			dist := cs.Clusters[a].Centroid.Distance(cs.Clusters[b].Centroid)
			if dist < cs.mergeThreshold {
				cs.Clusters[a].FaceIDs = append(cs.Clusters[a].FaceIDs, cs.Clusters[b].FaceIDs...)
				cs.Clusters[a].Centroid = cs.updateCentroid(
					cs.Clusters[a].Centroid,
					cs.Clusters[b].Centroid,
					len(cs.Clusters[a].FaceIDs),
				)
				delete(cs.Clusters, b)
				merged[b] = true

				events = append(events, MergeEvent{A: a, B: b, Result: a})
			}
		}
	}
	return events
}

// updateCentroid recalculates the average embedding after adding a new face.
// The centroid drifts gradually as more faces are added, this drift is what 
// allows the cluster to follow a face across years of change.
func (cs *ClusterStore) updateCentroid(current Embedding, incoming Embedding, totalFaces int) Embedding {
	updated := make([]float64, len(current.Values))
	weight := 1.0 / float64(totalFaces)

	for i := range current.Values {
		updated[i] = current.Values[i]*(1-weight) + incoming.Values[i]*weight
	}

	return Embedding{Values: updated}
}

func (cs *ClusterStore) newClusterID() string {
	cs.nextID++
	return fmt.Sprintf("cluster_%03d", cs.nextID)
}