package main

// Decision is the final output of the pipeline
type Decision struct {
	AccountID string
	RiskScore float64
	Action    string // "no_action", "observe", "suspend"
	Reasons   []string
}

// decide maps a risk score to a concrete action.
//
// The two thresholds encode the "observe vs act" split from the architecture:
//
//	Below 0.50 -> no action. Not enough signal to warrant attention.
//	0.50–0.74 -> observe. Confident enough that something is wrong,
//	but keep watching to map the full network before acting.
//	Suspending one account alerts operators and they adjust.
//	0.75+ -> suspend. High confidence. Take down the entire
//	detected cluster simultaneously so operators can't learn
//	which specific behavior triggered detection.
func decide(accountID string, riskScore float64, reasons []string) Decision {
	action := "no_action"
	switch {
	case riskScore >= 0.75:
		action = "suspend"
	case riskScore >= 0.50:
		action = "observe"
	}

	return Decision{
		AccountID: accountID,
		RiskScore: riskScore,
		Action:    action,
		Reasons:   reasons,
	}
}
