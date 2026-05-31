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

var repos []string

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
	rootCmd.PersistentFlags().StringSliceVarP(&repos, "repo", "R", nil, "Select another repository using the [HOST/]OWNER/REPO format (comma-separated for multiple)")

	carapace.Gen(rootCmd)
	carapace.Gen(rootCmd).FlagCompletion(carapace.ActionMap{
		"repo": actions.ActionRepos().MultiParts("/").UniqueList(","),
	})

	spec.AddMacro("Repos", spec.MacroN(actions.ActionRepos))
	spec.Register(rootCmd)
}

func resolveRepos() ([]repository.Repository, error) {
	if len(repos) > 0 {
		result := make([]repository.Repository, 0, len(repos))
		for _, r := range repos {
			parsed, err := repository.Parse(r)
			if err != nil {
				return nil, fmt.Errorf("failed to parse repo %q: %w", r, err)
			}
			result = append(result, parsed)
		}
		return result, nil
	}
	current, err := repository.Current()
	if err != nil {
		return nil, err
	}
	return []repository.Repository{current}, nil
}

func Execute() error {
	return rootCmd.Execute()
}
