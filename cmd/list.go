package cmd

import (
	"fmt"

	"github.com/carapace-sh/carapace"
	"github.com/rsteube/gh-slop/pkg/slop"
	"github.com/spf13/cobra"
)

var minContributions int

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List pull requests from new contributors",
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := resolveRepo()
		if err != nil {
			return err
		}

		prs, err := slop.ListNewContributors(r, minContributions)
		if err != nil {
			return err
		}

		for _, pr := range prs {
			fmt.Printf("#%d\t%s\t%s\n", pr.Number, pr.Author, pr.Title)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().IntVarP(&minContributions, "min-contributions", "m", 1, "Minimum merged PRs for a contributor to be filtered out")
	carapace.Gen(listCmd)
}
