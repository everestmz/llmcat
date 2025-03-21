package llmcat

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/everestmz/llmcat/ctxspec"
	"github.com/everestmz/llmcat/git"
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
	log.Debug().Str("path", filename).Strs("symbols", options.ExpandSymbols).Msg("Expanding file with symbols")
	outputLines := []string{}

	options.SetDefaults()

	lines := strings.Split(text, "\n")
	totalLines := len(lines)

	gutterWidth := len(fmt.Sprint(len(lines))) + 1 // add 1 line for a space before the separator

	// Calculate page bounds
	startIndex := options.StartLine - 1
	endIndex := totalLines

	if options.PageSize > 0 {
		endIndex = min(startIndex+options.PageSize, totalLines)
	}

	// Validate bounds
	if startIndex >= totalLines {
		startIndex = max(totalLines-1, 0)
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
		return slices.Contains(options.ExpandSymbols, chunk.Name)
	}

	chunks, err := treesym.GetSymbols(context.TODO(), &treesym.SourceFile{
		Path: filename,
		Text: text,
	})
	outline := chunks.GetOutline()
	if err == language.ErrUnsupportedExtension || len(outline) == 0 {
		// Just print all the lines within the range
		for lineIndex, line := range lines[startIndex:endIndex] {
			lineNum := lineIndex + 1
			outputLines = append(outputLines, addLineInfo(line, startIndex, lineNum))
		}
	} else if err != nil {
		return "", err
	} else {
		for _, chunk := range outline {
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

	if rdo.IncludeExtensions != nil && rdo.ExcludeExtensions != nil {
		return fmt.Errorf("cannot specify extensions to inlcude and exclude")
	}

	for i := range len(rdo.IncludeExtensions) {
		if !strings.HasPrefix(rdo.IncludeExtensions[i], ".") {
			rdo.IncludeExtensions[i] = "." + rdo.IncludeExtensions[i]
		}
	}

	for i := range len(rdo.ExcludeExtensions) {
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

	walkFilesFunc := func(path string, info os.FileInfo, err error) error {
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

		if info.IsDir() {
			return nil
		}

		extension := filepath.Ext(path)

		if slices.Contains(options.ExcludeExtensions, extension) {
			log.Debug().Str("file", path).Msgf("Excluding because extension matches excludes")
			return nil
		}

		if len(options.IncludeExtensions) > 0 {
			if !slices.Contains(options.IncludeExtensions, extension) {
				log.Debug().Str("file", path).Msgf("Excluding because extension is not included")
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

		relPath := path
		if filepath.IsAbs(path) {
			relPath, err = filepath.Rel(dirName, path)
			if err != nil {
				return err
			}
		}

		fileOpts := options.FileOptions
		if spec, ok := options.ContextSpec[relPath]; ok {
			fileOptions := fileOpts.Copy()
			if len(spec.Symbols) > 0 {
				fileOpts.ExpandSymbols = spec.Symbols
			} else {
				// Just show everything
				fileOptions.Outline = false
			}
		}
		rendered, err := RenderFile(relPath, string(text), fileOpts)
		if err != nil {
			return fmt.Errorf("error rendering file %s: %w", relPath, err)
		}
		files = append(files, rendered)

		return nil
	}

	repoRoot, isGitRepo := git.FindRepoRoot(".")

	if options.ContextSpec != nil {
		for path := range options.ContextSpec {
			info, err := os.Stat(path)
			if err != nil {
				return "", fmt.Errorf("unable to stat file (%s) in context spec: %w", path, err)
			}

			err = walkFilesFunc(path, info, nil)
			if err != nil {
				return "", err
			}
		}
	} else if isGitRepo {
		repo, err := git.NewRepo(repoRoot)
		if err != nil {
			return "", err
		}

		relativeToRoot, err := filepath.Rel(repoRoot, dirName)
		if err != nil {
			return "", err
		}

		err = repo.LsFilesFunc(relativeToRoot, func(f *git.File) error {
			info, err := os.Stat(f.Name)
			if err != nil {
				return err
			}

			return walkFilesFunc(f.Name, info, nil)
		}, &git.LsFilesOptions{
			// TODO: maybe we make this an option the user can pass in?
			IncludeUntrackedFiles: true,
		})

		if err != nil {
			return "", err
		}
	} else {
		err = filepath.WalkDir(dirName, func(path string, d fs.DirEntry, err error) error {
			info, err := d.Info()
			if err != nil {
				return err
			}

			return walkFilesFunc(path, info, err)
		})
	}

	if err != nil {
		return "", err
	}

	return strings.Join(files, "\n\n"), nil
}
