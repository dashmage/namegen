package gen

import (
	"strings"
	"sync"
	"testing"
)

func TestRandomNameContainingKeepsSubstringInternal(t *testing.T) {
	SetSeed(42)
	for range 50 {
		name := RandomNameContaining(8, "OrA")
		start := strings.Index(name, "ora")
		if len(name) != 8 || start <= 0 || start+len("ora") >= len(name) {
			t.Fatalf("RandomNameContaining(8, %q) = %q, want lowercase substring with a character on each side", "OrA", name)
		}
	}
}

func TestRandomNameContainingRejectsInvalidInput(t *testing.T) {
	if got := RandomNameContaining(4, "abc"); got != "" {
		t.Fatalf("RandomNameContaining with insufficient length = %q, want empty", got)
	}
	if got := RandomNameContaining(5, "a-b"); got != "" {
		t.Fatalf("RandomNameContaining with punctuation = %q, want empty", got)
	}
	if got := RandomNameContaining(5, "abc"); got != "" {
		t.Fatalf("RandomNameContaining with forbidden consonant adjacency = %q, want empty", got)
	}
}

func TestBuildRhythmPatternKeepsCompleteNuclei(t *testing.T) {
	SetSeed(84)
	for length := 2; length <= 24; length++ {
		for range 100 {
			pattern := buildRhythmPattern(length)
			if len(pattern) != length {
				t.Fatalf("buildRhythmPattern(%d) length = %d", length, len(pattern))
			}
			if !strings.Contains(string(pattern), "V") {
				t.Fatalf("buildRhythmPattern(%d) = %q, has no vowel nucleus", length, pattern)
			}
			for i := 2; i < len(pattern); i++ {
				if pattern[i-2] == 'V' && pattern[i-1] == 'V' && pattern[i] == 'V' {
					t.Fatalf("buildRhythmPattern(%d) = %q, contains a triple-vowel run", length, pattern)
				}
			}
		}
	}
}

func TestWeightedTemplateDoesNotLeavePartialSyllable(t *testing.T) {
	SetSeed(19)
	for remaining := 2; remaining <= 30; remaining++ {
		template := weightedTemplate(remaining)
		remainder := remaining - len(template)
		if remainder < 0 || remainder == 1 {
			t.Fatalf("template %q leaves invalid remainder %d from %d", template, remainder, remaining)
		}
	}
}

func TestGenerateIncludesRequiredSubstring(t *testing.T) {
	SetSeed(42)
	result := Generate(Options{
		MaxAttempts: 1000,
		Count:       3,
		Length:      5,
		Substring:   "ora",
		Threshold:   -100,
	})

	if len(result.Names) != 3 {
		t.Fatalf("generated names = %d, want 3; attempts = %d", len(result.Names), result.Attempts)
	}
	for _, name := range result.Names {
		if len(name.Name) != 5 || !strings.Contains(name.Name, "ora") {
			t.Errorf("generated name %q does not satisfy length and substring constraints", name.Name)
		}
	}
}

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
