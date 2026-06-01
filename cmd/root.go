package cmd

import (
	"github.com/carapace-sh/carapace"
	spec "github.com/carapace-sh/carapace-spec"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/rsteube/gh-slop/pkg/actions"
	"github.com/rsteube/gh-slop/pkg/crush"
	"github.com/rsteube/gh-slop/pkg/slop"
	"github.com/spf13/cobra"
)

var repos []string

var rootCmd = &cobra.Command{
	Use:   "gh-slop",
	Short: "A gh extension to handle slop contributions",
	Args:  cobra.NoArgs,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return crush.Run(cmd.Context())
	},
}

func init() {
	rootCmd.PersistentFlags().StringSliceVarP(&repos, "repo", "R", nil, "Select another repository using the [HOST/]OWNER/REPO format (comma-separated for multiple)")

	carapace.Gen(rootCmd)
	carapace.Gen(rootCmd).FlagCompletion(carapace.ActionMap{
		"repo": actions.ActionRepos().MultiParts("/").UniqueList(","),
	})

	spec.AddMacro("Repos", spec.MacroN(actions.ActionRepos))
	spec.AddMacro("Sloppers", spec.MacroV(actions.ActionSloppers))
	spec.AddMacro("SlopperPRs", spec.MacroI(actions.ActionSlopperPRs))
	spec.Register(rootCmd)
}

func ResolveRepos() ([]repository.Repository, error) {
	return slop.ResolveRepos(repos)
}

func Execute() error {
	return rootCmd.Execute()
}
