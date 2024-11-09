package main

import (
	"fmt"
	"os"

	"github.com/everestmz/llmcat"
	"github.com/spf13/cobra"
)

func main() {
	var options llmcat.RenderFileOptions
	var rootCmd = &cobra.Command{
		Use:   "llmcat [path]",
		Short: "Display file contents with optional line numbers and formatting",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]

			fileInfo, err := os.Stat(path)
			if err != nil {
				return fmt.Errorf("error accessing path: %v", err)
			}

			if fileInfo.IsDir() {
				dirOptions := &llmcat.RenderDirectoryOptions{
					FileOptions: &options,
				}
				output, err := llmcat.RenderDirectory(path, dirOptions)
				if err != nil {
					return fmt.Errorf("error processing directory: %v", err)
				}
				fmt.Println(output)
			} else {
				content, err := os.ReadFile(path)
				if err != nil {
					return fmt.Errorf("error reading file: %v", err)
				}
				output := llmcat.RenderFile(path, string(content), &options)
				fmt.Println(output)
			}

			return nil
		},
	}

	// Add flags
	flags := rootCmd.Flags()
	flags.BoolVarP(&options.OutputMarkdown, "markdown", "m", true, "output in markdown format")
	flags.BoolVarP(&options.ShowLineNumbers, "line-numbers", "n", true, "show line numbers")
	flags.StringVarP(&options.GutterSeparator, "separator", "s", "|", "gutter separator character")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
