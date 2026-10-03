package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/Viswesh-G/scope/internal/navigation"
	"github.com/spf13/cobra"
)

var (
	refsName string
	refsPath string
	refsFile string
	refsLine int
	refsJSON bool
)

var refsCmd = &cobra.Command{
	Use:   "refs",
	Short: "Find type-aware references to a Go symbol",
	Long: `Find references to a Go declaration using Go package type information.

If a name is declared more than once under the selected path, use --file and
--line to select the exact declaration. This command type-checks packages and
reports package errors rather than guessing from matching identifier text.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		results, err := navigation.FindReferences(navigation.Config{
			Path: refsPath,
			Name: refsName,
			File: refsFile,
			Line: refsLine,
		})
		if err != nil {
			return err
		}
		if refsJSON {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(results)
		}

		for _, result := range results {
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", result.Kind, result.Name)
			fmt.Fprintf(cmd.OutOrStdout(), "  definition: %s:%d:%d (%s)\n",
				result.Definition.File, result.Definition.Line, result.Definition.Column, result.Definition.Package)
			fmt.Fprintf(cmd.OutOrStdout(), "  references: %d\n", len(result.References))
			for _, reference := range result.References {
				fmt.Fprintf(cmd.OutOrStdout(), "    %s:%d:%d (%s)\n",
					reference.File, reference.Line, reference.Column, reference.Package)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(refsCmd)
	flags := refsCmd.Flags()
	flags.StringVarP(&refsName, "name", "n", "", "exact Go symbol name")
	flags.StringVarP(&refsPath, "path", "p", ".", "Go module or package tree to search")
	flags.StringVarP(&refsFile, "file", "f", "", "source file containing the declaration")
	flags.IntVar(&refsLine, "line", 0, "line containing the declaration (requires --file)")
	flags.BoolVar(&refsJSON, "json", false, "emit definition and references as JSON")
	_ = refsCmd.MarkFlagRequired("name")
	_ = refsCmd.MarkFlagFilename("file", ".go")
}
