package main

import "fmt"

// Weights reflect real-world signal reliability

const (
	weightBehavioral = 0.20
	weightTiming     = 0.35
	weightGraph      = 0.30
	weightContent    = 0.15
	signalThreshold  = 0.70
)

// Signal holds the output of each parallel analysis
// Each field is a score from 0.0 (clean) to 1.0 (suspicious)
type Signal struct {
	Behavioral float64
	Timing     float64
	Graph      float64
	Content    float64
}

// score combines the four signals into a single weighted risk score.
func score(sig Signal) (float64, []string) {
	total := sig.Behavioral*weightBehavioral +
		sig.Timing*weightTiming +
		sig.Graph*weightGraph +
		sig.Content*weightContent

	var reasons []string
	if sig.Behavioral > signalThreshold {
		reasons = append(reasons, fmt.Sprintf("suspicious posting regularity (%.2f)", sig.Behavioral))
	}
	if sig.Timing > signalThreshold {
		reasons = append(reasons, fmt.Sprintf("synchronized timing with other accounts (%.2f)", sig.Timing))
	}
	if sig.Graph > signalThreshold {
		reasons = append(reasons, fmt.Sprintf("island-shaped follower graph (%.2f)", sig.Graph))
	}
	if sig.Content > signalThreshold {
		reasons = append(reasons, fmt.Sprintf("near-duplicate content detected (%.2f)", sig.Content))
	}

	return total, reasons
}
