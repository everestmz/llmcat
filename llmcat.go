package llmcat

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/everestmz/llmcat/ctxspec"
	"github.com/everestmz/llmcat/treesym"
	"github.com/everestmz/llmcat/treesym/language"
	"github.com/gobwas/glob"
	"github.com/rs/zerolog/log"
)

type RenderFileOptions struct {
	Outline         bool     `json:"outline"`
	OutputMarkdown  bool     `json:"output_markdown"`
	ShowLineNumbers bool     `json:"hide_line_numbers"`
	GutterSeparator string   `json:"gutter_separator"`
	PageSize        int      `json:"page_size"`
	StartLine       int      `json:"start_line"`
	ShowPageInfo    bool     `json:"show_page_info"`
	ExpandSymbols   []string `json:"expand_symbols"`
}

// TODO: split this up so we produce another type which contains
// ExpandSymbols - they're not a generic input, they're specific to a file
func (ro *RenderFileOptions) Copy() *RenderFileOptions {
	new := *ro

	return &new
}

func (ro *RenderFileOptions) SetDefaults() {
	if ro.GutterSeparator == "" {
		ro.GutterSeparator = "|"
	}

	if ro.PageSize == 0 {
		ro.PageSize = 10000
	}

	if ro.StartLine < 1 {
		ro.StartLine = 1
	}
}

func RenderFile(filename, text string, options *RenderFileOptions) (string, error) {
	outputLines := []string{}

	options.SetDefaults()

	lines := strings.Split(text, "\n")
	totalLines := len(lines)

	gutterWidth := len(fmt.Sprint(len(lines))) + 1 // add 1 line for a space before the separator

	// Calculate page bounds
	startIndex := options.StartLine - 1
	endIndex := totalLines

	if options.PageSize > 0 {
		endIndex = startIndex + options.PageSize
		if endIndex > totalLines {
			endIndex = totalLines
		}
	}

	// Validate bounds
	if startIndex >= totalLines {
		startIndex = totalLines - 1
		if startIndex < 0 {
			startIndex = 0
		}
		endIndex = totalLines
	}

	if options.OutputMarkdown {
		header := fmt.Sprintf("```%s", filename)
		if options.ShowPageInfo && options.PageSize > 0 {
			header += fmt.Sprintf(" (Lines %d-%d of %d)", startIndex+1, endIndex, totalLines)
		}
		outputLines = append(outputLines, header)
	}

	if startIndex > 0 {
		marker := fmt.Sprintf("... (%d lines above) ...", startIndex)
		if options.ShowLineNumbers {
			marker = fmt.Sprintf("%s%s %s", strings.Repeat(" ", gutterWidth), options.GutterSeparator, marker)
		}
		outputLines = append(outputLines, marker)
	}

	addLineInfo := func(line string, offset, lineIndex int) string {
		if options.ShowLineNumbers {
			lineNum := offset + lineIndex
			padding := strings.Repeat(" ", gutterWidth-len(fmt.Sprint(lineNum)))
			line = fmt.Sprintf("%d%s%s %s", lineNum, padding, options.GutterSeparator, line)
		}

		return line
	}

	shouldExpandChunk := func(chunk *treesym.OutlineChunk) bool {
		for _, symbolName := range options.ExpandSymbols {
			if chunk.Name == symbolName {
				return true
			}
		}

		return false
	}

	chunks, err := treesym.GetSymbols(context.TODO(), &treesym.SourceFile{
		Path: filename,
		Text: text,
	})
	if err == language.ErrUnsupportedExtension {
		// Just print all the lines within the range
		for lineIndex, line := range lines[startIndex:endIndex] {
			lineNum := lineIndex + 1
			outputLines = append(outputLines, addLineInfo(line, startIndex, lineNum))
		}
	} else if err != nil {
		return "", err
	} else {
		for _, chunk := range chunks.GetOutline() {
			// Tree-sitter rows are 0-indexed, our line numbers are 1-indexed
			startLine := chunk.StartRow + 1
			endLine := chunk.EndRow + 1

			if endLine < startIndex {
				continue
			}

			if startLine > endIndex {
				continue
			}

			// This chunk is at least partially in the range

			if options.Outline && chunk.ShouldOmit && !shouldExpandChunk(chunk) {
				// Specify how many lines have been omitted (it may not be the size of the chunk,
				// if some of it is on the next or previous page!)
				var headLinesAlreadyOmitted, tailLinesAlreadyOmitted int
				if startLine < startIndex {
					headLinesAlreadyOmitted = startIndex - startLine
				}
				if endLine > endIndex {
					tailLinesAlreadyOmitted = endLine - endIndex
				}

				// The line numbers are inclusive - so add one
				omittedLine := fmt.Sprintf("... (%d lines omitted) ...", (endLine-startLine+1)-headLinesAlreadyOmitted-tailLinesAlreadyOmitted)

				if options.ShowLineNumbers {
					padding := strings.Repeat(" ", gutterWidth)
					omittedLine = fmt.Sprintf("%s%s %s", padding, options.GutterSeparator, omittedLine)
				}

				outputLines = append(outputLines, omittedLine)
			} else {
				lines := strings.Split(chunk.Content, "\n")

				for lineNum, line := range lines {
					outputLines = append(outputLines, addLineInfo(line, startLine, lineNum))
				}
			}
		}
	}

	if endIndex < totalLines {
		marker := fmt.Sprintf("... (%d lines below) ...", totalLines-endIndex)
		if options.ShowLineNumbers {
			marker = fmt.Sprintf("%s%s %s", strings.Repeat(" ", gutterWidth), options.GutterSeparator, marker)
		}
		outputLines = append(outputLines, marker)
	}

	if options.OutputMarkdown {
		outputLines = append(outputLines, "```")
	}

	return strings.Join(outputLines, "\n"), nil
}

// We should probably allow for glob-based ignores, extension-based ignores, and some other dir-based filters
type RenderDirectoryOptions struct {
	FileOptions       *RenderFileOptions  `json:"file_options"`
	IgnoreGlobs       []string            `json:"ignore_globs"`
	IncludeGlobs      []string            `json:"include_globs"`
	IncludeExtensions []string            `json:"include_extensions"`
	ExcludeExtensions []string            `json:"exclude_extensions"`
	ContextSpec       ctxspec.ContextSpec `json:"context_spec"`

	compiledIgnoreGlobs  []glob.Glob
	compiledIncludeGlobs []glob.Glob
}

func (rdo *RenderDirectoryOptions) SetDefaults() error {
	rdo.FileOptions.SetDefaults()

	rdo.IgnoreGlobs = append(rdo.IgnoreGlobs, "**/.git/**")

	if rdo.IncludeExtensions != nil && rdo.ExcludeExtensions != nil {
		return fmt.Errorf("cannot specify extensions to inlcude and exclude")
	}

	for i := 0; i < len(rdo.IncludeExtensions); i++ {
		if !strings.HasPrefix(rdo.IncludeExtensions[i], ".") {
			rdo.IncludeExtensions[i] = "." + rdo.IncludeExtensions[i]
		}
	}

	for i := 0; i < len(rdo.ExcludeExtensions); i++ {
		if !strings.HasPrefix(rdo.ExcludeExtensions[i], ".") {
			rdo.ExcludeExtensions[i] = "." + rdo.ExcludeExtensions[i]
		}
	}

	for _, ignoreGlob := range rdo.IgnoreGlobs {
		g, err := glob.Compile(ignoreGlob)
		if err != nil {
			return err
		}

		rdo.compiledIgnoreGlobs = append(rdo.compiledIgnoreGlobs, g)
	}

	for _, includeGlob := range rdo.IncludeGlobs {
		g, err := glob.Compile(includeGlob)
		if err != nil {
			return err
		}

		rdo.compiledIncludeGlobs = append(rdo.compiledIncludeGlobs, g)
	}

	return nil
}

func RenderDirectory(dirName string, options *RenderDirectoryOptions) (string, error) {
	var files []string

	err := options.SetDefaults()
	if err != nil {
		return "", err
	}

	dirName, err = filepath.Abs(dirName)
	if err != nil {
		return "", err
	}

	err = filepath.WalkDir(dirName, func(path string, d fs.DirEntry, err error) error {
		for _, ignoreGlob := range options.compiledIgnoreGlobs {
			if ignoreGlob.Match(path) {
				log.Debug().Str("file", path).Str("glob", fmt.Sprint(ignoreGlob)).Msgf("Ignored file")
				return nil
			}
		}

		if len(options.compiledIncludeGlobs) > 0 {
			include := false
			for _, includeGlob := range options.compiledIncludeGlobs {
				if includeGlob.Match(path) {
					log.Debug().Str("file", path).Str("glob", fmt.Sprint(includeGlob)).Msgf("Included file")
					include = true
				}
			}

			if !include {
				return nil
			}
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		extension := filepath.Ext(path)

		for _, ext := range options.ExcludeExtensions {
			if extension == ext {
				return nil
			}
		}

		if len(options.IncludeExtensions) > 0 {
			included := false
			for _, ext := range options.IncludeExtensions {
				if extension == ext {
					included = true
				}
			}

			if !included {
				return nil
			}
		}

		// Check if file has execute permission using file mode bits
		if info.Mode()&0111 != 0 {
			return nil
		}

		text, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dirName, path)
		if err != nil {
			return err
		}

		fileOpts := options.FileOptions
		if spec, ok := options.ContextSpec[relPath]; ok {
			fileOpts = fileOpts.Copy()
			if len(spec.Symbols) > 0 {
				fileOpts.ExpandSymbols = spec.Symbols
			} else {
				// Just show everything
				fileOpts.Outline = false
			}
		}
		rendered, err := RenderFile(relPath, string(text), fileOpts)
		if err != nil {
			return fmt.Errorf("error rendering file %s: %w", relPath, err)
		}
		files = append(files, rendered)

		return nil
	})

	if err != nil {
		return "", err
	}

	return strings.Join(files, "\n\n"), nil
}
