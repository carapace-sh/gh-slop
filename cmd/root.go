package cmd

import (
	"github.com/carapace-sh/carapace"
	spec "github.com/carapace-sh/carapace-spec"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/rsteube/gh-slop/pkg/actions"
	"github.com/spf13/cobra"
)

var repos []string

var rootCmd = &cobra.Command{
	Use:   "gh-slop",
	Short: "A gh extension to handle slop contributions",
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Run: func(cmd *cobra.Command, args []string) {},
}

func init() {
	rootCmd.PersistentFlags().StringSliceVarP(&repos, "repo", "R", nil, "Select another repository using the [HOST/]OWNER/REPO format (comma-separated for multiple)")

	carapace.Gen(rootCmd)
	carapace.Gen(rootCmd).FlagCompletion(carapace.ActionMap{
		"repo": actions.ActionRepos().MultiParts("/").UniqueList(","),
	})

	spec.AddMacro("Repos", spec.MacroN(actions.ActionRepos))
	spec.AddMacro("Sloppers", spec.MacroV(actions.ActionSloppers))
	spec.Register(rootCmd)
}

func ResolveRepos() ([]repository.Repository, error) {
	return actions.ResolveRepos(repos)
}

func Execute() error {
	return rootCmd.Execute()
}
