package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/carapace-sh/carapace"
	"github.com/rsteube/gh-slop/pkg/actions"
	"github.com/rsteube/gh-slop/pkg/render"
	"github.com/rsteube/gh-slop/pkg/slop"
	"github.com/spf13/cobra"
)

var closeCmd = &cobra.Command{
	Use:   "close [slopper]",
	Short: "Close all open PRs from a given slopper",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repos, err := ResolveRepos()
		if err != nil {
			return err
		}

		slopper := args[0]
		prs, err := slop.FindPRsByAuthor(repos, slopper)
		if err != nil {
			return err
		}

		if len(prs) == 0 {
			fmt.Println("No open PRs found for", slopper)
			return nil
		}

		fmt.Println(render.Render(prs))
		fmt.Printf("\nClose %d PR(s) from @%s? [y/N] ", len(prs), slopper)

		reader := bufio.NewReader(os.Stdin)
		resp, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		if !strings.EqualFold(strings.TrimSpace(resp), "y") {
			fmt.Println("Cancelled.")
			return nil
		}

		prRefs := make([]string, len(prs))
		for i, pr := range prs {
			prRefs[i] = pr.PullRequest.Ref(pr.Repo)
		}

		results, err := slop.ClosePRs(prRefs)
		if err != nil {
			return err
		}

		for _, r := range results {
			fmt.Printf("%s: %s\n", r.Ref(), r.State)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(closeCmd)
	carapace.Gen(closeCmd)
	closeCmd.Flags().StringSliceVarP(&repos, "repo", "R", nil, "Repository (owner/repo)")
	carapace.Gen(closeCmd).FlagCompletion(carapace.ActionMap{
		"repo": actions.ActionRepos().MultiParts("/").UniqueList(","),
	})
	carapace.Gen(closeCmd).PositionalCompletion(
		carapace.ActionCallback(func(c carapace.Context) carapace.Action {
			return actions.ActionSloppers(repos...)
		}),
	)
}
