package treesym

import (
	"context"
	"fmt"
	"testing"
)

const pythonSample = `
AIDER_SITE_URL = "https://aider.chat"
AIDER_APP_NAME = "Aider"

os.environ["OR_SITE_URL"] = AIDER_SITE_URL
os.environ["OR_APP_NAME"] = AIDER_APP_NAME
os.environ["LITELLM_MODE"] = "PRODUCTION"

# 'import litellm' takes 1.5 seconds, defer it!

class LazyLiteLLM:
    _lazy_module = None

    def __getattr__(
    	self,
    	name,
    ):
        if name == "_lazy_module":
            return super()
        self._load_litellm()
        return getattr(self._lazy_module, name)

    def _load_litellm(self):
        if self._lazy_module is not None:
            return

        self._lazy_module = importlib.import_module("litellm")

        self._lazy_module.suppress_debug_info = True
        self._lazy_module.set_verbose = False
        self._lazy_module.drop_params = True
        self._lazy_module._logging._disable_debugging()

litellm = LazyLiteLLM()

__all__ = [litellm]
`

const goExample = `package treesym

import (
	"context"
	"fmt"

	"github.com/everestmz/llmcat/treesym/language"
	"github.com/everestmz/llmcat/treesym/tags"
	sitter "github.com/smacker/go-tree-sitter"
)

type SourceFile struct {
	Path string
	Text string
}

func GetSymbols(ctx context.Context, file *SourceFile) (any, error) {
	tsLang, err := GetTreeSitterLanguage(file.Path)
	if err != nil {
		return nil, err
	}

	parser, err := GetParser(file.Path)
	if err != nil {
		return nil, err
	}

	tree, err := parser.ParseCtx(ctx, nil, []byte(file.Text))
	if err != nil {
		return nil, err
	}

	lang, err := language.GetLanguage(file.Path)
	if err != nil {
		return nil, err
	}

	tagsQuery, err := tags.GetTagsQuery(lang)
	if err != nil {
		return nil, err
	}

	q, err := sitter.NewQuery([]byte(tagsQuery), tsLang)
	if err != nil {
		return nil, err
	}

	qc := sitter.NewQueryCursor()

	qc.Exec(q, tree.RootNode())

	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}

		m = qc.FilterPredicates(m, []byte(file.Text))

		for _, cap := range m.Captures {
			captureName := q.CaptureNameForId(cap.Index)
			fmt.Println("-----------")
			fmt.Println("Capture", captureName)
			fmt.Println(cap.Node.Content([]byte(file.Text)))
		}
	}

	return nil, nil
}`

func TestGetSymbols(t *testing.T) {
	goProc, err := GetSymbols(context.TODO(), &SourceFile{
		Path: "treesym/treesym.go",
		Text: goExample,
	})
	if err != nil {
		t.Fatalf(err.Error())
	}

	pythonProc, err := GetSymbols(context.TODO(), &SourceFile{
		Path: "treesym/test.py",
		Text: pythonSample,
	})
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name     string
		kind     string
		summary  string
		language string
	}{
		{
			name: "SourceFile",
			kind: "type",
			summary: `SourceFile struct {
Path string
Text string
}`,
			language: "go",
		},
		{
			name:     "GetSymbols",
			kind:     "function",
			summary:  "func GetSymbols(ctx context.Context, file *SourceFile) (any, error) {",
			language: "go",
		},
		{
			name:     "LazyLiteLLM",
			kind:     "class",
			summary:  "", // don't check class since no summarization
			language: "python",
		},
		{
			name: "__getattr__",
			kind: "function",
			summary: `def __getattr__(
    	self,
    	name,
    ):`,
			language: "python",
		},
		{
			name:     "_load_litellm",
			kind:     "function",
			summary:  "def _load_litellm(self):",
			language: "python",
		},
	}

	for i, sym := range append(goProc.Definitions, pythonProc.Definitions...) {
		test := tests[i]

		if sym.Name != test.name {
			t.Fatalf("test %d name %s != %s", i, test.name, sym.Name)
		}

		if sym.Kind != test.kind {
			t.Fatalf("test %d kind %s != %s", i, test.kind, sym.Kind)
		}

		if !(sym.Kind == "function" || sym.Kind == "method") {
			continue
		}

		if sym.Summary != test.summary {
			t.Fatalf("test %d summary %s != %s", i, test.summary, sym.Summary)
		}
	}

	// TODO: actual tests for GetOutline
	// These print statements show it works for these simple examples though
	for _, chunk := range goProc.GetOutline() {
		fmt.Println(chunk.ShouldOmit, chunk.StartRow, ":", chunk.EndRow)
		fmt.Println(chunk.Content)
	}

	for _, chunk := range pythonProc.GetOutline() {
		fmt.Println(chunk.ShouldOmit, chunk.StartRow, ":", chunk.EndRow)
		fmt.Println(chunk.Content)
	}
}
