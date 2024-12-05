package language

import (
	"errors"
)

type Language string

var ErrUnsupportedExtension = errors.New("unsupported file extension")

const (
	Python     Language = "python"
	Javascript Language = "javascript"
	Typescript Language = "typescript"
	Tsx        Language = "tsx"
	Go         Language = "go"
	Rust       Language = "rust"
	Cpp        Language = "c++"
	C          Language = "c"
	Ruby       Language = "ruby"
	Java       Language = "java"
)

func GetLanguage(extension string) (Language, error) {
	switch extension {
	case ".py":
		return Python, nil
	case ".js":
		fallthrough
		// NOTE: we're doing this because the javascript tags are being weird, 
		// maybe because the JS tree sitter grammar is out of date, or the tags just don't work
		// return Javascript, nil
	case ".ts":
		return Typescript, nil
	case ".jsx", ".tsx":
		return Tsx, nil
	case ".go":
		return Go, nil
	case ".rs":
		return Rust, nil
	case ".cpp", ".cc", ".hpp", ".h", ".cxx":
		return Cpp, nil
	case ".rb":
		return Ruby, nil
	case ".java":
		return Java, nil
	case ".c":
		return C, nil
	default:
		return "", ErrUnsupportedExtension
	}
}
