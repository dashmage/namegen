package cli

import "testing"

func TestValidate(t *testing.T) {
	validSubstring := NewConfig(10, 5, 5, 1, false, false, false, 80)
	validSubstring.Substring = "AbC"
	missingExplicitLength := NewConfig(10, 5, 5, 1, false, false, false, 80)
	missingExplicitLength.LengthProvided = false
	missingExplicitLength.Substring = "abc"
	shortSubstringLength := NewConfig(10, 5, 4, 1, false, false, false, 80)
	shortSubstringLength.Substring = "abc"
	invalidSubstring := NewConfig(10, 5, 5, 1, false, false, false, 80)
	invalidSubstring.Substring = "a-b"

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "valid",
			config:  NewConfig(10, 5, 6, 1, false, false, false, 80),
			wantErr: false,
		},
		{
			name:    "rejects non positive attempts",
			config:  NewConfig(0, 5, 6, 1, false, false, false, 80),
			wantErr: true,
		},
		{
			name:    "rejects non positive count",
			config:  NewConfig(10, 0, 6, 1, false, false, false, 80),
			wantErr: true,
		},
		{
			name:    "rejects non positive length",
			config:  NewConfig(10, 5, 0, 1, false, false, false, 80),
			wantErr: true,
		},
		{
			name:    "accepts substring with sufficient explicitly provided length",
			config:  validSubstring,
			wantErr: false,
		},
		{
			name:    "requires explicit length with substring",
			config:  missingExplicitLength,
			wantErr: true,
		},
		{
			name:    "requires two extra characters beyond substring",
			config:  shortSubstringLength,
			wantErr: true,
		},
		{
			name:    "rejects non ascii letters in substring",
			config:  invalidSubstring,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
