package main

import (
	"math"
	"sort"
	"sync"
	"time"
)

type result struct {
	field string
	score float64
}

type analysis struct {
	name string
	fn   func() float64
}

// collectSignals fans out into four goroutines. One per signal type
// and waits for all of them before returning. This mirros how a real
// system would run these analyses concurrently rather than sequentially.
func collectSignals(account Account, action Action, recentActors []Action) Signal {
	var (
		wg  sync.WaitGroup
		sig Signal
	)

	results := make(chan result, 4)

	analyses := []analysis{
		{"behavioral", func() float64 { return analyzeBehavioral(account) }},
		{"timing", func() float64 { return analyzeTiming(action, recentActors) }},
		{"graph", func() float64 { return analyzeGraph(account) }},
		{"content", func() float64 { return analyzeContent(account.RecentText) }},
	}

	for _, a := range analyses {
		wg.Add(1)
		go func(a analysis) {
			defer wg.Done()
			results <- result{a.name, a.fn()}
		}(a)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for r := range results {
		switch r.field {
		case "behavioral":
			sig.Behavioral = r.score
		case "timing":
			sig.Timing = r.score
		case "graph":
			sig.Graph = r.score
		case "content":
			sig.Content = r.score
		}
	}

	return sig
}

// analyzeBehavioral checks how regular an account's posting intervals are.
//
// Humans post irregularly except for case where posts are scheduled
// Bots tend toward suspiciously regular interval even when they add jitter.
// We use a coefficient of variation (stddev/mean) as the regularity measure:
//
// CV (stddev/mean) was chosen because it is scale invariant. It normalizes the
// standard deviation by the mean, so the regularity score does not depend on the
// absolute size of the intervals.
//
// For example:
//
// Account A posts every 10 seconds, stddev of 1 second
// Account B posts every 10 hours, stddev of 1 hour
//
// Both have the same CV of 0.10, but Raw stddev would say Account B is far more irregular.
// which is why we are not using Raw stddev or variance.
//
// Another option is "Autocorrelation"
// It measures whether intervals repeat in a pattern.
// More powerful than CV for detecting bots that vary
// their intervals in a repeating cycle.
// But significantly more complex to implement and explain.
func analyzeBehavioral(account Account) float64 {
	if len(account.PostTimes) < 5 {
		return 0.1
	}

	intervals := make([]float64, len(account.PostTimes)-1)
	for i := 1; i < len(account.PostTimes); i++ {
		intervals[i-1] = float64(account.PostTimes[i].Sub(account.PostTimes[i-1]).Seconds())
	}

	mean := 0.0
	for _, v := range intervals {
		mean += v
	}

	mean /= float64(len(intervals))

	variance := 0.0
	for _, v := range intervals {
		d := v - mean
		variance += d * d
	}

	stddev := math.Sqrt(variance / float64(len(intervals)))

	cv := stddev / mean
	regularityScore := 1.0 - math.Min(cv, 1.0)

	oddHourPosts := 0
	for _, t := range account.PostTimes {
		h := t.UTC().Hour()
		if h >= 2 && h <= 5 {
			oddHourPosts++
		}
	}
	oddHourRatio := float64(oddHourPosts) / float64(len(account.PostTimes))

	return regularityScore*0.6 + oddHourRatio*0.4

}

// analyzeTiming checks whether many accounts acted on the same target
// within an unnaturally tight time window
func analyzeTiming(action Action, recentActors []Action) float64 {
	var timestamps []time.Time
	for _, a := range recentActors {
		if a.TargetID == action.TargetID && a.Type == action.Type {
			timestamps = append(timestamps, a.Timestamp)
		}
	}

	if len(timestamps) < 10 {
		return 0.0
	}

	sort.Slice(timestamps, func(i, j int) bool {
		return timestamps[i].Before(timestamps[j])
	})

	gaps := make([]float64, len(timestamps)-1)
	for i := 1; i < len(timestamps); i++ {
		gaps[i-1] = float64(timestamps[i].Sub(timestamps[i-1]).Milliseconds())
	}

	mean := 0.0
	for _, g := range gaps {
		mean += g
	}
	mean /= float64(len(gaps))

	variance := 0.0
	for _, g := range gaps {
		d := g - mean
		variance += d * d
	}
	stddev := math.Sqrt(variance / float64(len(gaps)))

	cv := stddev / mean
	return 1.0 - math.Min(cv/2.0, 1.0)
}

// analyzeGraph checks whether an account sits inside a bot cluster
//
// Bot networks follow each other to build legitimacy, creating a dense
// subgraph weakly connected to the rest of the platform.
// Real communities are dense internally but have many bridges outward
func analyzeGraph(account Account) float64 {
	if len(account.Followers) == 0 {
		return 0.5
	}

	followerSet := make(map[string]bool, len(account.Followers))
	for _, f := range account.Followers {
		followerSet[f] = true
	}

	internalEdges := 0
	for _, f := range account.Following {
		if followerSet[f] {
			internalEdges++
		}
	}

	totalConnections := len(followerSet) + len(account.Following)
	if totalConnections == 0 {
		return 0.5
	}

	internalRatio := float64(internalEdges) / float64(totalConnections)

	accountAgeDays := time.Since(account.CreatedAt).Hours() / 24
	followerVelocity := 0.0
	if accountAgeDays > 0 {
		followerVelocity = float64(len(account.Followers)) / accountAgeDays
	}

	velocityScore := math.Min(followerVelocity/50.0, 1.0)

	return internalRatio*0.7 + velocityScore*0.3
}

// analyzeContent checks for near-duplicate posts using Jaccard similarity
// on character bigrams. This catches bots that slightly vary their text to
// evade exact string matching.
func analyzeContent(posts []string) float64 {
	if len(posts) < 2 {
		return 0.0
	}

	totalPairs := 0
	similarPairs := 0

	for i := 0; i < len(posts); i++ {
		for j := i + 1; j < len(posts); j++ {
			if jaccardSimilarity(posts[i], posts[j]) > 0.7 {
				similarPairs++
			}
			totalPairs++
		}
	}

	if totalPairs == 0 {
		return 0.0
	}

	return float64(similarPairs) / float64(totalPairs)
}

// jaccardSimilarity computes overlap between two strings using character
// bigrams. "hello" -> {"he", "el", "ll", "lo"}. Score of 1.0 = identical.
func jaccardSimilarity(a, b string) float64 {
	setA := bigrams(a)
	setB := bigrams(b)

	intersection := 0
	for k := range setA {
		if setB[k] {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 1.0
	}

	return float64(intersection) / float64(union)
}

func bigrams(s string) map[string]bool {
	set := make(map[string]bool)
	runes := []rune(s)

	for i := 0; i < len(runes)-1; i++ {
		set[string(runes[i:i+2])] = true
	}

	return set
}
