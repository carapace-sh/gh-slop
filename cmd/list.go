package cmd

import (
	"fmt"

	"github.com/carapace-sh/carapace"
	"github.com/rsteube/gh-slop/pkg/render"
	"github.com/rsteube/gh-slop/pkg/slop"
	"github.com/spf13/cobra"
)

var minContributions int

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List pull requests from new contributors",
	RunE: func(cmd *cobra.Command, args []string) error {
		repos, err := resolveRepos()
		if err != nil {
			return err
		}

		prs, err := slop.ListNewContributors(repos, minContributions)
		if err != nil {
			return err
		}

		fmt.Println(render.Render(prs))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().IntVarP(&minContributions, "min-contributions", "m", 1, "Minimum merged PRs for a contributor to be filtered out")
	carapace.Gen(listCmd)
}
