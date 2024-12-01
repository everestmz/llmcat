package ctxspec

import (
	"testing"
)

func TestParseContextSpec(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []*FileContextSpec
		wantErr bool
	}{
		{
			name:  "empty input",
			input: "",
			want:  nil, // Change this from empty slice to nil
		},
		{
			name:  "single file no symbols",
			input: "main.go",
			want: []*FileContextSpec{
				{Filename: "main.go"},
			},
		},
		{
			name:  "single file with symbols",
			input: `main.go MyFunc AnotherFunc`,
			want: []*FileContextSpec{
				{
					Filename: "main.go",
					Symbols:  []string{"MyFunc", "AnotherFunc"},
				},
			},
		},
		{
			name: "multiple files with symbols",
			input: `main.go MyFunc
parser.go Parse ParseLine`,
			want: []*FileContextSpec{
				{
					Filename: "main.go",
					Symbols:  []string{"MyFunc"},
				},
				{
					Filename: "parser.go",
					Symbols:  []string{"Parse", "ParseLine"},
				},
			},
		},
		{
			name: "quoted strings with spaces",
			input: `"main file.go" "My Function"
"complex parser.go" "Parse Items"`,
			want: []*FileContextSpec{
				{
					Filename: "main file.go",
					Symbols:  []string{"My Function"},
				},
				{
					Filename: "complex parser.go",
					Symbols:  []string{"Parse Items"},
				},
			},
		},
		{
			name:    "unclosed quote",
			input:   `main.go "unclosed`,
			wantErr: true,
		},
		{
			name:  "escaped quotes",
			input: `main.go "Method \"quoted\" name"`,
			want: []*FileContextSpec{
				{
					Filename: "main.go",
					Symbols:  []string{`Method "quoted" name`},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseContextSpec(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseContextSpec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if (got == nil) != (tt.want == nil) {
					t.Errorf("ParseContextSpec() nil mismatch: got = %v, want = %v", got, tt.want)
					return
				}
				if len(got) != len(tt.want) {
					t.Errorf("ParseContextSpec() length mismatch: got = %v, want = %v", got, tt.want)
					return
				}
				for i := range got {
					if got[i].Filename != tt.want[i].Filename {
						t.Errorf("ParseContextSpec() filename mismatch at index %d: got = %q, want = %q", i, got[i].Filename, tt.want[i].Filename)
					}
					if !slicesEqual(got[i].Symbols, tt.want[i].Symbols) {
						t.Errorf("ParseContextSpec() symbols mismatch at index %d: got = %v, want = %v", i, got[i].Symbols, tt.want[i].Symbols)
					}
				}
			}
		})
	}
}

// Helper function to compare slices
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
