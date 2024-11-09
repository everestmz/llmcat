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
}

func (ro *RenderFileOptions) SetDefaults() {
	if ro.GutterSeparator == "" {
		gs := "|"

		ro.GutterSeparator = gs
	}
}

func RenderFile(filename, text string, options *RenderFileOptions) string {
	outputLines := []string{}

	options.SetDefaults()

	if options.OutputMarkdown {
		outputLines = append(outputLines, "```"+filename)
	}

	lines := strings.Split(text, "\n")

	gutterWidth := len(fmt.Sprint(len(lines))) + 1 // add 1 line for a space before the separator

	for i, line := range lines {
		if options.ShowLineNumbers {
			lineNum := i + 1
			padding := strings.Repeat(" ", gutterWidth-len(fmt.Sprint(lineNum)))
			line = fmt.Sprintf("%d%s%s %s", lineNum, padding, options.GutterSeparator, line)
		}

		outputLines = append(outputLines, line)
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
