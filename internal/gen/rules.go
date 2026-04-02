package gen

import (
	"strings"

	"github.com/dashmage/namegen/internal/defaults"
)

var (
	// Keep only non-consonant-adjacency pairs here.
	// Consonant-consonant restrictions are handled by IllegalConsonantAdjacency.
	UncommonSequences = []string{"yb", "yj", "yf", "jj", "vv", "qq", "xx", "zz", "iq", "iy", "uq", "vf", "wh", "yh", "jh", "qh", "ii", "uu", "hwl", "dzd", "gfm", "ynk", "fdd", "zdd", "ddl", "lzd", "vdd", "wlk"}

	// Missing keys mean no explicit adjacency restriction for that consonant.
	AllowedNextConsonants = map[byte]string{
		'b': "lr",
		'c': "hlrstw",
		'd': "lrw",
		'f': "klrst",
		'g': "hlnrsw",
		'h': "grsw",
		'j': "",
		'k': "hsr",
		'p': "hlrstw",
		'q': "",
		't': "hlrsw",
		'v': "lrv",
		'w': "r",
		'x': "pst",
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

type Rule struct {
	Name        string
	Description string
	Penalty     int
	Check       func(string) bool
}

var HardRules = []Rule{
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

var SoftRules = []Rule{
	{
		Name:        "uncommon_sequence",
		Description: "Penalizes impossible or very awkward letter pairs.",
		Penalty:     25,
		Check:       UncommonSequence,
	},
	{
		Name:        "q_not_followed_by_u",
		Description: "Penalizes q when it is not followed by u.",
		Penalty:     10,
		Check:       QWithoutU,
	},
	{
		Name:        "rare_letter_density",
		Description: "Penalizes words with too many rare letters (j/q/x/z).",
		Penalty:     20,
		Check:       RareLetterDensity,
	},
	{
		Name:        "repeated_same_vowel_pair",
		Description: "Penalizes doubled identical vowels that often sound awkward.",
		Penalty:     10,
		Check:       RepeatedSameVowelPair,
	},
	{
		Name:        "double_consonant_ending",
		Description: "Penalizes words ending in doubled consonants.",
		Penalty:     10,
		Check:       DoubleConsonantEnding,
	},
}

// ThreeConsecutiveConsonants returns true if 3 or more consecutive consonants are present.
func ThreeConsecutiveConsonants(word string) bool {
	var counter int
	for i := range len(word) {
		if !isVowel(word[i]) {
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

// IllegalEnding returns true if a word ends with impossible letters
func IllegalEnding(word string) bool {
	if len(word) == 0 {
		return false
	}
	return strings.Contains(defaults.IllegalEndingChars, string(word[len(word)-1]))
}

// UncommonSequence returns true if a word contains a rare sequence of letters
func UncommonSequence(word string) bool {
	for _, seq := range UncommonSequences {
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

// RepeatedSameVowelPair returns true if identical vowels repeat back-to-back.
func RepeatedSameVowelPair(word string) bool {
	for i := 1; i < len(word); i++ {
		if word[i] != word[i-1] {
			continue
		}
		if isVowel(word[i]) {
			return true
		}
	}
	return false
}

// DoubleConsonantEnding returns true when the word ends with a repeated consonant.
func DoubleConsonantEnding(word string) bool {
	if len(word) < 2 {
		return false
	}
	last := word[len(word)-1]
	prev := word[len(word)-2]
	return last == prev && !isVowel(last)
}

// IllegalConsonantAdjacency returns true if a consonant pair violates directional allow-lists.
func IllegalConsonantAdjacency(word string) bool {
	for i := 0; i < len(word)-1; i++ {
		left := word[i]
		right := word[i+1]

		if isVowel(left) || isVowel(right) {
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
