package gen

import "testing"

func TestGenerateReturnsUniqueNames(t *testing.T) {
	SetSeed(42)
	result := Generate(Options{
		MaxAttempts: 1000,
		Count:       10,
		Length:      1,
		Threshold:   -100,
		TuneEnabled: true,
	})

	if result.RequestedCount != 10 {
		t.Fatalf("RequestedCount = %d, want 10", result.RequestedCount)
	}

	seen := make(map[string]struct{}, len(result.Names))
	for _, name := range result.Names {
		if _, duplicate := seen[name.Name]; duplicate {
			t.Fatalf("Generate returned duplicate name %q", name.Name)
		}
		seen[name.Name] = struct{}{}
	}

	if result.DuplicateRejects == 0 {
		t.Fatal("expected repeated candidates to be counted as duplicate rejects")
	}
	if len(result.AttemptLog) != result.Attempts {
		t.Fatalf("attempt log length = %d, want %d", len(result.AttemptLog), result.Attempts)
	}

	loggedDuplicate := false
	for _, attempt := range result.AttemptLog {
		if attempt.RejectReason == "duplicate" {
			loggedDuplicate = true
			break
		}
	}
	if !loggedDuplicate {
		t.Fatal("expected duplicate candidate rejection in attempt log")
	}
}
