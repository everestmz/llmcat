package treesym

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/everestmz/llmcat/treesym/language"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/c"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/ruby"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/typescript/tsx"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

func GetTreeSitterLanguage(path string) (*sitter.Language, error) {
	// Get the file extension
	ext := strings.ToLower(filepath.Ext(path))

	lang, err := language.GetLanguage(ext)
	if err != nil {
		return nil, err
	}

	// Set the appropriate language based on file extension
	switch lang {
	case language.Python:
		return python.GetLanguage(), nil
	case language.Javascript:
		return javascript.GetLanguage(), nil
	case language.Typescript:
		return typescript.GetLanguage(), nil
	case language.Tsx:
		return tsx.GetLanguage(), nil
	case language.Go:
		return golang.GetLanguage(), nil
	case language.Rust:
		return rust.GetLanguage(), nil
	case language.Cpp:
		return cpp.GetLanguage(), nil
	case language.Ruby:
		return ruby.GetLanguage(), nil
	case language.Java:
		return java.GetLanguage(), nil
	case language.C:
		return c.GetLanguage(), nil
	default:
		return nil, fmt.Errorf("unsupported language: %s", lang)
	}
}

// GetParser returns a tree-sitter parser based on the file extension
func GetParser(path string) (*sitter.Parser, error) {
	// Create a new parser
	parser := sitter.NewParser()

	l, err := GetTreeSitterLanguage(path)
	if err != nil {
		return nil, err
	}

	parser.SetLanguage(l)

	return parser, nil
}
