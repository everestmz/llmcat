package tags

import (
	_ "embed"
	"fmt"

	"github.com/everestmz/llmcat/treesym/language"
)

var (
	//go:embed c.scm
	CTags string

	//go:embed cpp.scm
	CppTags string

	//go:embed go.scm
    GoTags string

	//go:embed python.scm
	PythonTags string

	//go:embed ruby.scm
	RubyTags string

	//go:embed rust.scm
	RustTags string

	//go:embed typescript.scm
	TypescriptTags string

	//go:embed java.scm
	JavaTags string

	//go:embed javascript.scm
    JavascriptTags string
)

var queries = map[language.Language]string{
	language.Python: PythonTags,
	language.Javascript: JavascriptTags,
	language.C: CTags,
	language.Typescript: TypescriptTags,
	language.Tsx: TypescriptTags,
	language.Go: GoTags,
	language.Rust: RustTags,
	language.Cpp: CppTags,
	language.Ruby: RubyTags,
	language.Java: JavaTags,
}

func GetTagsQuery(lang language.Language) (string, error) {
	query, ok := queries[lang]
	if !ok {
		return "", fmt.Errorf("unsupported language: %s", lang)
	}
	return query, nil
}
