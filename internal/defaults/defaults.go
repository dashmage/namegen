package defaults

const (
	Threshold = 80

	MaxAttempts = 200
	Count       = 10
	Length      = 5

	BaseScore = 100
	BaseAlpha = 0.5

	Vowels     = "aeiouy"
	Consonants = "bcdfghjklmnpqrstvwxz"

	FinalConsonantBiasPercent = 35

	IllegalEndingChars = "qjvw"

	StartToken byte = '^'
	EndToken   byte = '$'
	VocabSize       = 28

	VeryLowProbCutoff = -4.2
	LowProbCutoff     = -3.6
	MidProbCutoff     = -3.1
	// Average log-probability where the full good-probability bonus is reached.
	GoodProbBonusCutoff = -2.6

	VeryLowProbPenalty = 30
	LowProbPenalty     = 20
	MidProbPenalty     = 15
	GoodProbBonus      = 5
)
