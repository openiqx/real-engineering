package main

import (
	"fmt"
	"time"
)

// Action represents anything a user does on the platform.
type Action struct {
	AccountID string
	Type      string // "like", "retweet", "follow"
	TargetID  string // tweet or account being acted on
	Timestamp time.Time
}

// Account holds everything we know about a user
type Account struct {
	ID         string
	CreatedAt  time.Time
	PostTimes  []time.Time
	Followers  []string
	Following  []string
	RecentText []string //recent post content
}

// runPipeline ties all three stages as seen in the architecture together
// collect signals -> score -> decide
func runPipeline(account Account, action Action, recentActors []Action) Decision {
	signals := collectSignals(account, action, recentActors)
	riskScore, reasons := score(signals)
	return decide(account.ID, riskScore, reasons)
}

func main() {
	now := time.Now()

	// Simulate a suspicious account:
	// 1. only 2 days old
	// 2. posts every 10 minutes exactly (suspiciously regular)
	// 3. all followers also follow back
	// 4. near-duplicate content with slight variations
	suspiciousAccount := Account{
		ID:        "acc_suspicious_001",
		CreatedAt: now.Add(-48 * time.Hour),
		PostTimes: []time.Time{
			now.Add(-10 * time.Minute),
			now.Add(-20 * time.Minute),
			now.Add(-30 * time.Minute),
			now.Add(-40 * time.Minute),
			now.Add(-50 * time.Minute),
			now.Add(-60 * time.Minute),
		},
		Followers: []string{"bot_002", "bot_003", "bot_004", "bot_005"},
		Following: []string{"bot_002", "bot_003", "bot_004", "bot_005"},
		RecentText: []string{
			"Buy cheap followers now click here",
			"Buy cheap follower's now, click here!",
			"Buy cheap folowers now click here",
		},
	}

	// The action that triggered this pipeline run.
	action := Action{
		AccountID: suspiciousAccount.ID,
		Type:      "like",
		TargetID:  "tweet_viral_001",
		Timestamp: now,
	}

	// Simulate a cluster of 20 accounts all liking the same tweet
	// within a 500ms window, unnaturally tight clustering.
	recentActors := make([]Action, 20)
	for i := range recentActors {
		recentActors[i] = Action{
			AccountID: fmt.Sprintf("bot_%03d", i),
			Type:      "like",
			TargetID:  "tweet_viral_001",
			Timestamp: now.Add(-time.Duration(i*25) * time.Millisecond),
		}
	}

	d := runPipeline(suspiciousAccount, action, recentActors)

	fmt.Printf("Account:    %s\n", d.AccountID)
	fmt.Printf("Risk score: %.2f\n", d.RiskScore)
	fmt.Printf("Action:     %s\n", d.Action)
	fmt.Println("Reasons:")
	for _, r := range d.Reasons {
		fmt.Printf("  - %s\n", r)
	}
}
