package ctxspec

import (
	"bufio"
	"fmt"
	"strings"
)

type ContextSpec map[string]*FileContextSpec

type FileContextSpec struct {
	Filename string
	Symbols  []string
}

func MergeContextSpecs(specs ...*FileContextSpec) ContextSpec {
	filenameToSpec := map[string]*FileContextSpec{}

	for _, spec := range specs {
		if len(spec.Symbols) == 0 {
			// Just specifying the file
			filenameToSpec[spec.Filename] = spec
			continue
		}

		if existing, ok := filenameToSpec[spec.Filename]; ok {
			if len(existing.Symbols) == 0 {
				// We've already selected the whole file
				continue
			}
			existing.Symbols = append(existing.Symbols, spec.Symbols...)
		} else {
			filenameToSpec[spec.Filename] = spec
		}
	}

	return filenameToSpec
}

func ParseContextSpec(contextDefinition string) (ContextSpec, error) {
	scanner := bufio.NewScanner(strings.NewReader(contextDefinition))
	scanner.Split(bufio.ScanLines)

	var items []*FileContextSpec

	for scanner.Scan() {
		line := scanner.Text()
		newItem, err := ParseSpecLine(line)
		if err != nil {
			return nil, fmt.Errorf("Error for line '%s': %w", line, err)
		}

		items = append(items, newItem)
	}

	return MergeContextSpecs(items...), nil
}

func ParseSpecLine(line string) (*FileContextSpec, error) {
	if line == "" {
		return nil, nil
	}

	parts, err := getLineParts(line)
	if err != nil {
		return nil, err
	}

	filename := parts[0]
	contextItem := &FileContextSpec{
		Filename: filename,
	}

	// Our options right now are a whole file, or a symbol.
	// Each row can have one filename, but multiple options for symbols
	if len(parts) == 1 {
		return contextItem, nil
	}

	// We have more than one item for this file
	for _, item := range parts[1:] {
		contextItem.Symbols = append(contextItem.Symbols, item)
	}

	return contextItem, nil
}

func getLineParts(line string) ([]string, error) {
	var parts []string

	var currentPart strings.Builder

	var inQuotes bool
	var escaped bool

	for _, char := range line {
		switch {
		case escaped:
			currentPart.WriteRune(char)
			escaped = false
		case char == '\\':
			escaped = true
		case char == '"' && !escaped:
			inQuotes = !inQuotes

			if !inQuotes {
				parts = append(parts, currentPart.String())
				currentPart.Reset()
			}

		case char == ' ' && !inQuotes:
			if currentPart.Len() > 0 {
				parts = append(parts, currentPart.String())
				currentPart.Reset()
			}
		default:
			currentPart.WriteRune(char)
		}
	}

	if currentPart.Len() > 0 {
		parts = append(parts, currentPart.String())
	}

	if inQuotes {
		return nil, fmt.Errorf("Found quote with no matching closing quote")
	}

	return parts, nil
}
