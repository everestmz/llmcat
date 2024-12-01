# ctxspec

A parser for context specifications that define which parts of source files to include in LLM context windows. Supports both full-file and symbol-level granularity.

## Usage

Specify files and symbols to include:

```go
//Include specific functions from main.go and the entire config.go file

specs, err := ctxspec.ParseContextSpec(`
main.go ProcessRequest HandleError
config.go
`)

// Quotes for items containing spaces
specs, err := ctxspec.ParseContextSpec(`
"complex file.go" "Process Request"
utils.go "error handling"
`)
```

## Syntax

Each line follows the format:

```
copyfilename [symbol1 symbol2 ...]
```

- First item: filename
- Subsequent items (optional): symbols to include from that file
- Use quotes for items with spaces: "my file.go" "My Function"
- Escape quotes in strings: file.go "Function \"name\""

The parser automatically merges multiple specifications for the same file:

```go
// These get merged:
main.go Func1
main.go Func2

// Equivalent to:
main.go Func1 Func2

// But specifying the whole file takes precedence:
main.go         // Whole file
main.go Func1   // Ignored - whole file already selected
```
