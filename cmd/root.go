package cmd

import (
	"fmt"

	"github.com/carapace-sh/carapace"
	spec "github.com/carapace-sh/carapace-spec"
	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/rsteube/gh-slop/pkg/actions"
	"github.com/spf13/cobra"
)

var repo string

var rootCmd = &cobra.Command{
	Use:   "gh-slop",
	Short: "A gh extension to handle slop contributions",
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Run: func(cmd *cobra.Command, args []string) {
		client, err := api.DefaultRESTClient()
		if err != nil {
			fmt.Println(err)
			return
		}
		response := struct{ Login string }{}
		err = client.Get("user", &response)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("running as %s\n", response.Login)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&repo, "repo", "R", "", "Select another repository using the [HOST/]OWNER/REPO format")

	carapace.Gen(rootCmd)
	carapace.Gen(rootCmd).FlagCompletion(carapace.ActionMap{
		"repo": actions.ActionRepos().MultiParts("/"),
	})

	spec.AddMacro("Repos", spec.MacroN(actions.ActionRepos))
	spec.Register(rootCmd)
}

func resolveRepo() (repository.Repository, error) {
	if repo != "" {
		return repository.Parse(repo)
	}
	return repository.Current()
}

func Execute() error {
	return rootCmd.Execute()
}
