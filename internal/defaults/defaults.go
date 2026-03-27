package defaults

const (
	AcceptThreshold = 90

	CLIAttemptsDefault = 200
	CLICountDefault    = 10
	CLILengthDefault   = 5

	BaseScore = 100
	BaseAlpha = 0.5

	Vowels     = "aeiouy"
	Consonants = "bcdfghjklmnpqrstvwxyz"

	FinalConsonantBiasPercent = 35

	IllegalEndingChars = "qjvhw"

	StartToken byte = '^'
	EndToken   byte = '$'
	VocabSize       = 28

	VeryLowProbCutoff = -4.2
	LowProbCutoff     = -3.6
	MidProbCutoff     = -3.1

	VeryLowProbPenalty = 30
	LowProbPenalty     = 15
	MidProbPenalty     = 5
	GoodProbBonus      = 5
)
