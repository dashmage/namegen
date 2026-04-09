package gen

// AcceptedName stores an accepted generated name and its scoring metadata.
type AcceptedName struct {
	Name            string
	Score           int
	ProbabilityBand ProbabilityBand
}

// Result contains accepted names, aggregate counters, and optional candidate details.
type Result struct {
	Names           []AcceptedName
	AttemptLog      []Attempt
	Attempts        int
	HardRejects     int
	LowScoreRejects int
	Threshold       int
	RuleHits        RuleHits
}

// Attempt records scoring and rejection context for one candidate.
type Attempt struct {
	Candidate        string
	Score            int
	Threshold        int
	Accepted         bool
	RejectReason     string
	HardRule         string
	SoftRules        []Rule
	ProbabilityBand  ProbabilityBand
	AvgLogProb       float64
	BigramAdjustment int
}

// HardRuleStats returns the non-zero hard rules that were triggered during generation.

func (r Result) HardRuleStats() []Rule {
	stats := make([]Rule, 0, len(HardRules))
	for _, rule := range HardRules {
		hits := r.RuleHits.Hard[rule.Name]
		if hits == 0 {
			continue
		}
		stats = append(stats, Rule{
			Type:        rule.Type,
			Name:        rule.Name,
			Hits:        hits,
			Description: rule.Description,
		})
	}
	return stats
}

// SoftRuleStats returns the non-zero soft rules that were triggered during generation.

func (r Result) SoftRuleStats() []Rule {
	stats := make([]Rule, 0, len(SoftRules))
	for _, rule := range SoftRules {
		hits := r.RuleHits.Soft[rule.Name]
		if hits == 0 {
			continue
		}
		stats = append(stats, Rule{
			Type:        rule.Type,
			Name:        rule.Name,
			Hits:        hits,
			Penalty:     rule.Penalty,
			Description: rule.Description,
		})
	}
	return stats
}
