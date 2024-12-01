package ctxspec

import (
	"testing"
)

func TestParseContextSpec(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ContextSpec
		wantErr bool
	}{
		{
			name:  "empty input",
			input: "",
			want:  ContextSpec{},
		},
		{
			name:  "single file no symbols",
			input: "main.go",
			want: ContextSpec{
				"main.go": {Filename: "main.go"},
			},
		},
		{
			name:  "single file with symbols",
			input: `main.go MyFunc AnotherFunc`,
			want: ContextSpec{
				"main.go": {
					Filename: "main.go",
					Symbols:  []string{"MyFunc", "AnotherFunc"},
				},
			},
		},
		{
			name: "multiple files with symbols",
			input: `main.go MyFunc
parser.go Parse ParseLine`,
			want: ContextSpec{
				"main.go": {
					Filename: "main.go",
					Symbols:  []string{"MyFunc"},
				},
				"parser.go": {
					Filename: "parser.go",
					Symbols:  []string{"Parse", "ParseLine"},
				},
			},
		},
		{
			name: "quoted strings with spaces",
			input: `"main file.go" "My Function"
"complex parser.go" "Parse Items"`,
			want: ContextSpec{
				"main file.go": {
					Filename: "main file.go",
					Symbols:  []string{"My Function"},
				},
				"complex parser.go": {
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
			want: ContextSpec{
				"main.go": {
					Filename: "main.go",
					Symbols:  []string{`Method "quoted" name`},
				},
			},
		},
		{
			name: "merge symbols for same file",
			input: `main.go Func1
main.go Func2`,
			want: ContextSpec{
				"main.go": {
					Filename: "main.go",
					Symbols:  []string{"Func1", "Func2"},
				},
			},
		},
		{
			name: "whole file overrides symbols",
			input: `main.go Func1
main.go`,
			want: ContextSpec{
				"main.go": {
					Filename: "main.go",
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
				if len(got) != len(tt.want) {
					t.Errorf("ParseContextSpec() returned %d items, want %d", len(got), len(tt.want))
					return
				}
				for filename, wantSpec := range tt.want {
					gotSpec, exists := got[filename]
					if !exists {
						t.Errorf("ParseContextSpec() missing expected file %q", filename)
						continue
					}
					if gotSpec.Filename != wantSpec.Filename {
						t.Errorf("ParseContextSpec() for file %q got filename = %q, want %q",
							filename, gotSpec.Filename, wantSpec.Filename)
					}
					if !slicesEqual(gotSpec.Symbols, wantSpec.Symbols) {
						t.Errorf("ParseContextSpec() for file %q got symbols = %v, want %v",
							filename, gotSpec.Symbols, wantSpec.Symbols)
					}
				}
			}
		})
	}
}

func TestMergeContextSpecs(t *testing.T) {
	tests := []struct {
		name  string
		specs []*FileContextSpec
		want  ContextSpec
	}{
		{
			name:  "empty input",
			specs: nil,
			want:  ContextSpec{},
		},
		{
			name: "merge file with symbols",
			specs: []*FileContextSpec{
				{Filename: "main.go", Symbols: []string{"Func1"}},
				{Filename: "main.go", Symbols: []string{"Func2"}},
			},
			want: ContextSpec{
				"main.go": {Filename: "main.go", Symbols: []string{"Func1", "Func2"}},
			},
		},
		{
			name: "whole file overrides symbols",
			specs: []*FileContextSpec{
				{Filename: "main.go", Symbols: []string{"Func1"}},
				{Filename: "main.go"},
			},
			want: ContextSpec{
				"main.go": {Filename: "main.go"},
			},
		},
		{
			name: "symbols after whole file are ignored",
			specs: []*FileContextSpec{
				{Filename: "main.go"},
				{Filename: "main.go", Symbols: []string{"Func1"}},
			},
			want: ContextSpec{
				"main.go": {Filename: "main.go"},
			},
		},
		{
			name: "multiple distinct files",
			specs: []*FileContextSpec{
				{Filename: "main.go", Symbols: []string{"Func1"}},
				{Filename: "parser.go", Symbols: []string{"Parse"}},
			},
			want: ContextSpec{
				"main.go":   {Filename: "main.go", Symbols: []string{"Func1"}},
				"parser.go": {Filename: "parser.go", Symbols: []string{"Parse"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeContextSpecs(tt.specs...)
			if len(got) != len(tt.want) {
				t.Errorf("MergeContextSpecs() returned %d items, want %d", len(got), len(tt.want))
				return
			}
			for filename, wantSpec := range tt.want {
				gotSpec, exists := got[filename]
				if !exists {
					t.Errorf("MergeContextSpecs() missing expected file %q", filename)
					continue
				}
				if gotSpec.Filename != wantSpec.Filename {
					t.Errorf("MergeContextSpecs() for file %q got filename = %q, want %q",
						filename, gotSpec.Filename, wantSpec.Filename)
				}
				if !slicesEqual(gotSpec.Symbols, wantSpec.Symbols) {
					t.Errorf("MergeContextSpecs() for file %q got symbols = %v, want %v",
						filename, gotSpec.Symbols, wantSpec.Symbols)
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
