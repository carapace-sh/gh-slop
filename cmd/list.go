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

		multiRepo := len(repos) > 1

		var allPRs []render.PRWithRepo
		for _, r := range repos {
			prs, err := slop.ListNewContributors(r, minContributions)
			if err != nil {
				return fmt.Errorf("%s/%s: %w", r.Owner, r.Name, err)
			}

			for _, pr := range prs {
				repoLabel := ""
				if multiRepo {
					repoLabel = fmt.Sprintf("%s/%s", r.Owner, r.Name)
				}
				allPRs = append(allPRs, render.PRWithRepo{
					PullRequest: pr,
					Repo:        repoLabel,
				})
			}
		}

		fmt.Println(render.Render(allPRs))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().IntVarP(&minContributions, "min-contributions", "m", 1, "Minimum merged PRs for a contributor to be filtered out")
	carapace.Gen(listCmd)
}