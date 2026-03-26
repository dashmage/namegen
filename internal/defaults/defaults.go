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

	NGramVeryLowCutoff = -4.2
	NGramLowCutoff     = -3.6
	NGramMidCutoff     = -3.1

	NGramVeryLowDelta = -30
	NGramLowDelta     = -15
	NGramMidDelta     = -5
	NGramGoodDelta    = 5
)
