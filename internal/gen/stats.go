package gen

// ScoredWord stores an accepted generated word and its scoring metadata.
type ScoredWord struct {
	Word            string
	Score           int
	ProbabilityBand ProbabilityBand
}

// RulePenalty describes a soft rule penalty applied during evaluation.
type RulePenalty struct {
	Name        string
	Penalty     int
	Description string
}

// RuleStat informs how often a rule was triggered in a run.
type RuleStat struct {
	Name        string
	Hits        int
	Penalty     int
	Description string
}

// GenStats tracks aggregate counters and rule-hit totals for a run.
type GenStats struct {
	Attempts        int
	Accepted        int
	HardRejects     int
	LowScoreRejects int
	Threshold       int
	RuleHits        RuleHits
}

// RunResult contains accepted words, run stats, and optional attempt details.
type RunResult struct {
	Words       []ScoredWord
	Stats       GenStats
	GenAttempts []GenAttempt
}

// GenAttempt records scoring and rejection context for one generation attempt.
type GenAttempt struct {
	Word             string
	Score            int
	Threshold        int
	Accepted         bool
	RejectReason     string
	HardRule         string
	SoftRules        []RulePenalty
	ProbabilityBand  ProbabilityBand
	AvgLogProb       float64
	BigramAdjustment int
}

// HardRuleStats returns the non-zero hard rules that were triggered during generation.
func (s GenStats) HardRuleStats() []RuleStat {
	stats := make([]RuleStat, 0, len(HardRules))
	for _, rule := range HardRules {
		hits := s.RuleHits.Hard[rule.Name]
		if hits == 0 {
			continue
		}
		stats = append(stats, RuleStat{
			Name:        rule.Name,
			Hits:        hits,
			Description: rule.Description,
		})
	}
	return stats
}

// SoftRuleStats returns the non-zero soft rules that were triggered during generation.
func (s GenStats) SoftRuleStats() []RuleStat {
	stats := make([]RuleStat, 0, len(SoftRules))
	for _, rule := range SoftRules {
		hits := s.RuleHits.Soft[rule.Name]
		if hits == 0 {
			continue
		}
		stats = append(stats, RuleStat{
			Name:        rule.Name,
			Hits:        hits,
			Penalty:     rule.Penalty,
			Description: rule.Description,
		})
	}
	return stats
}
