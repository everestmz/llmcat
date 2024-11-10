package llmcat

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/gobwas/glob"
)

type RenderFileOptions struct {
	OutputMarkdown  bool   `json:"output_markdown"`
	ShowLineNumbers bool   `json:"hide_line_numbers"`
	GutterSeparator string `json:"gutter_separator"`
	PageSize        int    `json:"page_size"`
	StartLine       int    `json:"start_line"`
	ShowPageInfo    bool   `json:"show_page_info"`
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

func RenderFile(filename, text string, options *RenderFileOptions) string {
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

	for i, line := range lines[startIndex:endIndex] {
		if options.ShowLineNumbers {
			lineNum := i + startIndex + 1
			padding := strings.Repeat(" ", gutterWidth-len(fmt.Sprint(lineNum)))
			line = fmt.Sprintf("%d%s%s %s", lineNum, padding, options.GutterSeparator, line)
		}

		outputLines = append(outputLines, line)
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

	return strings.Join(outputLines, "\n")
}

// We should probably allow for glob-based ignores, extension-based ignores, and some other dir-based filters
type RenderDirectoryOptions struct {
	FileOptions   *RenderFileOptions `json:"file_options"`
	IgnoreGlobs   []string           `json:"ignore_globs"`
	compiledGlobs []glob.Glob
}

func (rdo *RenderDirectoryOptions) SetDefaults() error {
	rdo.FileOptions.SetDefaults()

	rdo.IgnoreGlobs = append(rdo.IgnoreGlobs, "**/.git/**")

	for _, ignoreGlob := range rdo.IgnoreGlobs {
		g, err := glob.Compile(ignoreGlob)
		if err != nil {
			return err
		}

		rdo.compiledGlobs = append(rdo.compiledGlobs, g)
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
		for _, ignoreGlob := range options.compiledGlobs {
			if ignoreGlob.Match(path) {
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
		files = append(files, RenderFile(relPath, string(text), options.FileOptions))

		return nil
	})

	if err != nil {
		return "", err
	}

	return strings.Join(files, "\n\n"), nil
}
