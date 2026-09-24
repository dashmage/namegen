package gen

// HardRuleAudit summarizes how the configured hard rules treat a candidate set.
// Rule hit counts can overlap because each candidate is checked against every rule.
type HardRuleAudit struct {
	CandidateCount int
	AcceptedCount  int
	RejectedCount  int
	Rules          []HardRuleAuditEntry
}

type HardRuleAuditEntry struct {
	Name     string
	Hits     int
	Examples []string
}

// FilterHardRuleValidNames returns candidates that pass every hard rule and a
// report containing independent hit counts for each rule.
func FilterHardRuleValidNames(names []string, exampleLimit int) ([]string, HardRuleAudit) {
	if exampleLimit < 0 {
		exampleLimit = 0
	}

	audit := HardRuleAudit{
		CandidateCount: len(names),
		Rules:          make([]HardRuleAuditEntry, len(HardRules)),
	}
	for i, rule := range HardRules {
		audit.Rules[i] = HardRuleAuditEntry{Name: rule.Name}
	}

	accepted := make([]string, 0, len(names))
	for _, name := range names {
		rejected := false
		for i, rule := range HardRules {
			if !rule.Check(name) {
				continue
			}
			rejected = true
			audit.Rules[i].Hits++
			if len(audit.Rules[i].Examples) < exampleLimit {
				audit.Rules[i].Examples = append(audit.Rules[i].Examples, name)
			}
		}
		if rejected {
			audit.RejectedCount++
			continue
		}
		accepted = append(accepted, name)
	}
	audit.AcceptedCount = len(accepted)
	return accepted, audit
}
