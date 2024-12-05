package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/everestmz/llmcat"
	"github.com/everestmz/llmcat/ctxspec"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func main() {
	var options llmcat.RenderFileOptions
	var dirOptions llmcat.RenderDirectoryOptions

	var rootCmd = &cobra.Command{
		Use:   "llmcat [path]",
		Short: "Display file contents with optional line numbers and formatting",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]

			contextSpecLines, err := cmd.Flags().GetStringSlice("expand")
			if err != nil {
				return err
			}

			contextSpec, err := ctxspec.ParseContextSpec(strings.Join(contextSpecLines, "\n"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			dirOptions.ContextSpec = contextSpec

			dirOptions.FileOptions = &options

			if strings.HasSuffix(path, ".git") {
				output, err := llmcat.RenderGitRepo(path, &dirOptions)
				if err != nil {
					return fmt.Errorf("error processing repository: %w", err)
				}
				fmt.Println(output)
			} else {
				fileInfo, err := os.Stat(path)
				if err != nil {
					return fmt.Errorf("error accessing path: %v", err)
				}

				if fileInfo.IsDir() {
					output, err := llmcat.RenderDirectory(path, &dirOptions)
					if err != nil {
						return fmt.Errorf("error processing directory (%s): %v", path, err)
					}
					fmt.Println(output)
				} else {
					content, err := os.ReadFile(path)
					if err != nil {
						return fmt.Errorf("error reading file: %v", err)
					}
					output, err := llmcat.RenderFile(path, string(content), &options)
					if err != nil {
						return fmt.Errorf("error rendering file: %w", err)
					}
					fmt.Println(output)
				}
			}

			return nil
		},
	}

	// Add flags
	flags := rootCmd.Flags()

	// Dev flags
	debug := flags.Bool("debug", false, "Set this flag to enable debug logs")
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	// File rendering flags
	flags.BoolVarP(&options.OutputMarkdown, "markdown", "m", true, "output in markdown format")
	flags.BoolVarP(&options.ShowLineNumbers, "line-numbers", "n", true, "show line numbers")
	flags.StringVarP(&options.GutterSeparator, "separator", "s", "|", "gutter separator character")
	flags.BoolVar(&options.Outline, "outline", false, "produce an outline for supported source files using tree-sitter")
	flags.StringArrayVar(&options.ExpandSymbols, "symbols", nil, "specify symbols to expand when showing an outline")

	// Pagination flags
	flags.IntVarP(&options.PageSize, "page-size", "p", 10000, "number of lines to show (0 = show all)")
	flags.IntVar(&options.StartLine, "start-line", 1, "first line to show (1-based)")
	flags.BoolVar(&options.ShowPageInfo, "show-page-info", true, "show page information in header")

	// Directory flags
	flags.StringSliceVar(&dirOptions.IgnoreGlobs, "ignore", []string{"**/.git/**"}, "glob patterns to ignore")
	flags.StringSliceVar(&dirOptions.IncludeGlobs, "include", nil, "glob patterns to include")
	flags.StringSliceVar(&dirOptions.ExcludeExtensions, "exclude-ext", nil, "comma-separated list of file extensions to exclude")
	flags.StringSliceVar(&dirOptions.IncludeExtensions, "ext", nil, "comma-separated list of file extensions to include")

	flags.StringSliceP("expand", "e", nil, "symbols or files to expand when showing an outline, in ctxspec format")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
