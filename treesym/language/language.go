package language

import "fmt"

type Language string

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
		return Javascript, nil
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
		return "", fmt.Errorf("unsupported file extension: %s", extension)
	}
}
