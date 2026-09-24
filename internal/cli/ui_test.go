package cli

import (
	"bytes"
	"testing"

	"github.com/dashmage/namegen/internal/gen"
)

func TestPrintShortfallWarning(t *testing.T) {
	tests := []struct {
		name   string
		result gen.Result
		want   string
	}{
		{
			name: "reports incomplete result",
			result: gen.Result{
				RequestedCount: 3,
				Names:          []gen.AcceptedName{{Name: "lora"}},
				Attempts:       20,
			},
			want: "warning: generated 1 of 3 requested names after 20 total attempts; consider increasing --attempts or adjusting generation settings\n",
		},
		{
			name: "does not report complete result",
			result: gen.Result{
				RequestedCount: 1,
				Names:          []gen.AcceptedName{{Name: "lora"}},
				Attempts:       1,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			printShortfallWarning(&output, test.result)
			if got := output.String(); got != test.want {
				t.Fatalf("warning = %q, want %q", got, test.want)
			}
		})
	}
}
