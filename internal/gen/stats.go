package gen

type ScoredWord struct {
	Word  string
	Score int
}

type RuleStat struct {
	Name        string
	Hits        int
	Penalty     int
	Description string
}

type GenStats struct {
	Attempts        int
	Accepted        int
	HardRejects     int
	LowScoreRejects int
	Threshold       int
	RuleHits        RuleCounters
}

type RunResult struct {
	Words []ScoredWord
	Stats GenStats
}

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
