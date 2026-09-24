package gen

import (
	"sync"
	"testing"
)

func TestRandomNameAndSetSeedAreSafeConcurrently(t *testing.T) {
	var wait sync.WaitGroup
	for worker := range 8 {
		wait.Add(1)
		go func(worker int) {
			defer wait.Done()
			for iteration := range 100 {
				if worker%2 == 0 {
					SetSeed(int64(worker*100 + iteration))
					continue
				}
				if got := RandomName(5); len(got) != 5 {
					t.Errorf("RandomName(5) length = %d, want 5", len(got))
					return
				}
			}
		}(worker)
	}
	wait.Wait()
}

func TestGenerateBoundsInitialResultCapacity(t *testing.T) {
	result := Generate(Options{
		MaxAttempts: 1,
		Count:       1 << 30,
		Length:      5,
		Threshold:   -100,
	})

	if result.Attempts != 1 {
		t.Fatalf("Attempts = %d, want 1", result.Attempts)
	}
}

func TestGenerateValidatesBeforeAllocatingAttemptLog(t *testing.T) {
	result := Generate(Options{
		MaxAttempts: 1 << 30,
		Count:       0,
		Length:      5,
		TuneEnabled: true,
	})

	if result.Attempts != 0 || len(result.AttemptLog) != 0 {
		t.Fatalf("invalid run result = attempts %d, log length %d; want 0, 0", result.Attempts, len(result.AttemptLog))
	}
}

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
