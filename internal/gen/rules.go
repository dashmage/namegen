package gen

import (
	"strings"

	"github.com/dashmage/namegen/internal/defaults"
)

var (
	// Keep only non-consonant-adjacency pairs here.
	// Consonant-consonant restrictions are handled by IllegalConsonantAdjacency.
	AwkwardPairs            = []string{"yb", "yj"}
	AwkwardStarts           = []string{"ng", "pt", "kn", "pn", "tl", "sr"}
	AwkwardEnds             = []string{"wh", "yh", "jh", "qh", "ii", "uu"}
	RareDoubles             = []string{"jj", "vv", "qq", "xx", "zz"}
	UncommonEnds            = []string{"iq", "uq", "vf"}
	AwkwardTerminalClusters = []string{"hwl", "dzd", "gfm", "ynk", "fdd", "zdd", "ddl", "lzd", "vdd", "wlk"}

	// Missing keys mean no explicit adjacency restriction for that consonant.
	AllowedNextConsonants = map[byte]string{
		'b': "lr",
		'c': "hlrstw",
		'd': "lrw",
		'f': "lr",
		'g': "hlnrw",
		'h': "rw",
		'j': "",
		'k': "hsr",
		'l': "bcdfglmnpqrstvz",
		'm': "bcdfglmnpstvz",
		'n': "cdfglmnpqstvz",
		'p': "hlrstw",
		'q': "",
		'r': "bcdfglmnpqstvwxz",
		's': "bcdfghklmnpqrtvwxz",
		't': "hlrsw",
		'v': "lrv",
		'w': "r",
		'x': "cpst",
		'z': "lr",
	}
	AllowedPrevConsonants = buildAllowedPrevConsonants(AllowedNextConsonants)
)

// buildAllowedPrevConsonants inverts a next-consonant allow-list map.
//
// Input map format: left -> allowed right consonants.
// Output map format: right -> allowed left consonants.
//
// For example, if
//
//	AllowedNextConsonants['k'] = "hsr"
//
// Then, the inverted result includes:
//
//	AllowedPrevConsonants['h'] contains 'k'
//	AllowedPrevConsonants['s'] contains 'k'
//	AllowedPrevConsonants['r'] contains 'k'
//
// The returned strings are ordered by defaults.Consonants for deterministic
// behavior and stable debugging output.
func buildAllowedPrevConsonants(next map[byte]string) map[byte]string {
	allowed := make(map[byte]map[byte]bool)
	for left, rights := range next {
		for i := 0; i < len(rights); i++ {
			right := rights[i]
			if allowed[right] == nil {
				allowed[right] = make(map[byte]bool)
			}
			allowed[right][left] = true
		}
	}

	prev := make(map[byte]string, len(allowed))
	for right, leftSet := range allowed {
		var list strings.Builder
		for i := range len(defaults.Consonants) {
			left := defaults.Consonants[i]
			if leftSet[left] {
				list.WriteByte(left)
			}
		}
		prev[right] = list.String()
	}

	return prev
}

type SoftRule struct {
	Name        string
	Description string
	Penalty     int
	Check       func(string) bool
}

type HardRule struct {
	Name        string
	Description string
	Check       func(string) bool
}

var HardRules = []HardRule{
	{
		Name:        "three_consecutive_consonants",
		Description: "Rejects words with hard-to-pronounce 3-consonant runs.",
		Check:       ThreeConsecutiveConsonants,
	},
	{
		Name:        "illegal_ending",
		Description: "Rejects words ending in awkward letters.",
		Check:       IllegalEnding,
	},
	{
		Name:        "missing_core_vowel",
		Description: "Rejects words without any core vowel (a/e/i/o/u).",
		Check:       MissingCoreVowel,
	},
	{
		Name:        "triple_same_letter",
		Description: "Rejects any triple repeated letter sequence.",
		Check:       TripleSameLetter,
	},
	{
		Name:        "illegal_consonant_adjacency",
		Description: "Rejects disallowed consonant-to-consonant transitions.",
		Check:       IllegalConsonantAdjacency,
	},
}

var SoftRules = []SoftRule{
	{
		Name:        "awkward_sequence",
		Description: "Penalizes impossible or very awkward letter pairs.",
		Penalty:     25,
		Check:       AwkwardSequence,
	},
	{
		Name:        "q_not_followed_by_u",
		Description: "Penalizes q when it is not followed by u.",
		Penalty:     20,
		Check:       QWithoutU,
	},
	{
		Name:        "double_rare_letter",
		Description: "Penalizes doubled rare letters like jj, qq, or zz.",
		Penalty:     14,
		Check:       RepeatedRareLetter,
	},
	{
		Name:        "awkward_boundary_cluster",
		Description: "Penalizes uncommon start or end boundary clusters.",
		Penalty:     5,
		Check:       AwkwardBoundary,
	},
	{
		Name:        "rare_letter_density",
		Description: "Penalizes words with too many rare letters (j/q/x/z).",
		Penalty:     21,
		Check:       RareLetterDensity,
	},
	{
		Name:        "uncommon_ending",
		Description: "Penalizes endings that are uncommon in English-like names.",
		Penalty:     13,
		Check:       UncommonEnding,
	},
	{
		Name:        "repeated_same_vowel_pair",
		Description: "Penalizes doubled identical vowels that often sound awkward.",
		Penalty:     15,
		Check:       RepeatedSameVowelPair,
	},
	{
		Name:        "awkward_terminal_cluster",
		Description: "Penalizes harsh multi-letter ending clusters.",
		Penalty:     15,
		Check:       AwkwardTerminalCluster,
	},
	{
		Name:        "double_terminal_consonant",
		Description: "Penalizes words ending in doubled consonants.",
		Penalty:     12,
		Check:       DoubleTerminalConsonant,
	},
}

// ThreeConsecutiveConsonants returns true if 3 or more consecutive consonants are present.
func ThreeConsecutiveConsonants(word string) bool {
	var counter int
	for i := range len(word) {
		if !checkVowel(string(word[i])) {
			counter++
			if counter == 3 {
				return true
			}
		} else {
			counter = 0
		}
	}
	return false
}

// ConsecutiveConsonants returns true if 4 or more consecutive consonants are present
func ConsecutiveConsonants(word string) bool {
	var counter int
	for i := range len(word) {
		if !checkVowel(string(word[i])) {
			counter++
			if counter == 4 {
				return true
			}
		} else {
			counter = 0
		}
	}
	return false
}

// IllegalEnding returns true if a word ends with impossible letters
func IllegalEnding(word string) bool {
	return strings.Contains(defaults.IllegalEndingChars, string(word[len(word)-1]))
}

// AwkwardSequence returns true if a word contains an impossible sequence of letters
func AwkwardSequence(word string) bool {
	for _, seq := range AwkwardPairs {
		if strings.Contains(word, seq) {
			return true
		}
	}
	return false
}

// QWithoutU returns true if q appears without a following u.
func QWithoutU(word string) bool {
	for i := 0; i < len(word); i++ {
		if word[i] != 'q' {
			continue
		}
		if i+1 >= len(word) || word[i+1] != 'u' {
			return true
		}
	}
	return false
}

// RepeatedRareLetter returns true if a rare double letter appears.
func RepeatedRareLetter(word string) bool {
	for _, seq := range RareDoubles {
		if strings.Contains(word, seq) {
			return true
		}
	}
	return false
}

// AwkwardBoundary returns true if a word has awkward start/end clusters.
func AwkwardBoundary(word string) bool {
	for _, prefix := range AwkwardStarts {
		if strings.HasPrefix(word, prefix) {
			return true
		}
	}
	for _, suffix := range AwkwardEnds {
		if strings.HasSuffix(word, suffix) {
			return true
		}
	}
	return false
}

// MissingCoreVowel returns true if no a/e/i/o/u appears in the word.
func MissingCoreVowel(word string) bool {
	return !strings.ContainsAny(word, "aeiou")
}

// TripleSameLetter returns true if any character repeats 3 times consecutively.
func TripleSameLetter(word string) bool {
	for i := 2; i < len(word); i++ {
		if word[i] == word[i-1] && word[i-1] == word[i-2] {
			return true
		}
	}
	return false
}

// RareLetterDensity returns true if 2+ of j/q/x/z are present.
func RareLetterDensity(word string) bool {
	rareCount := 0
	for i := 0; i < len(word); i++ {
		switch word[i] {
		case 'j', 'q', 'x', 'z':
			rareCount++
			if rareCount >= 2 {
				return true
			}
		}
	}
	return false
}

// UncommonEnding returns true if a word ends with an uncommon English-like suffix.
func UncommonEnding(word string) bool {
	for _, suffix := range UncommonEnds {
		if strings.HasSuffix(word, suffix) {
			return true
		}
	}
	return false
}

// RepeatedSameVowelPair returns true if identical vowels repeat back-to-back.
func RepeatedSameVowelPair(word string) bool {
	for i := 1; i < len(word); i++ {
		if word[i] != word[i-1] {
			continue
		}
		if checkVowel(string(word[i])) {
			return true
		}
	}
	return false
}

// AwkwardTerminalCluster returns true if a word ends with a known awkward terminal cluster.
func AwkwardTerminalCluster(word string) bool {
	for _, suffix := range AwkwardTerminalClusters {
		if strings.HasSuffix(word, suffix) {
			return true
		}
	}
	return false
}

// DoubleTerminalConsonant returns true when the word ends with a repeated consonant.
func DoubleTerminalConsonant(word string) bool {
	if len(word) < 2 {
		return false
	}
	last := word[len(word)-1]
	prev := word[len(word)-2]
	return last == prev && !checkVowel(string(last))
}

// IllegalConsonantAdjacency returns true if a consonant pair violates directional allow-lists.
func IllegalConsonantAdjacency(word string) bool {
	for i := 0; i < len(word)-1; i++ {
		left := word[i]
		right := word[i+1]

		if checkVowel(string(left)) || checkVowel(string(right)) {
			continue
		}

		if allowed, ok := AllowedNextConsonants[left]; ok && !strings.ContainsRune(allowed, rune(right)) {
			return true
		}

		if allowed, ok := AllowedPrevConsonants[right]; ok && !strings.ContainsRune(allowed, rune(left)) {
			return true
		}
	}

	return false
}
