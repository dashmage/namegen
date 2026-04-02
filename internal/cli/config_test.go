package cli

import "testing"

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  CLIConfig
		wantErr bool
	}{
		{
			name:    "valid",
			config:  NewCLIConfig(10, 5, 6, 1, false, false, false, 80),
			wantErr: false,
		},
		{
			name:    "rejects non positive attempts",
			config:  NewCLIConfig(0, 5, 6, 1, false, false, false, 80),
			wantErr: true,
		},
		{
			name:    "rejects non positive count",
			config:  NewCLIConfig(10, 0, 6, 1, false, false, false, 80),
			wantErr: true,
		},
		{
			name:    "rejects non positive length",
			config:  NewCLIConfig(10, 5, 0, 1, false, false, false, 80),
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
