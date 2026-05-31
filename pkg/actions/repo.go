package actions

import (
	"time"

	"github.com/carapace-sh/carapace"
	"github.com/rsteube/gh-slop/pkg/slop"
)

// ActionRepos completes repositories the current user can close pull requests in
//
//	owner/repo (description)
//	cli/cli (GitHub's official CLI)
func ActionRepos() carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		repos, err := slop.AccessibleRepos()
		if err != nil {
			return carapace.ActionMessage(err.Error())
		}

		var vals []string
		for _, r := range repos {
			vals = append(vals, r.Owner+"/"+r.Name, "")
		}

		return carapace.ActionValuesDescribed(vals...).Tag("repositories")
	}).Cache(24 * time.Hour)
}
