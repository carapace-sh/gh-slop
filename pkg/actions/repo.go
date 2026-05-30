package actions

import (
	"fmt"
	"strings"
	"time"

	"github.com/carapace-sh/carapace"
	"github.com/cli/go-gh/v2/pkg/api"
)

// ActionRepos completes repositories the current user can close pull requests in
//
//	owner/repo (description)
//	cli/cli (GitHub's official CLI)
func ActionRepos() carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		client, err := api.DefaultRESTClient()
		if err != nil {
			return carapace.ActionMessage(err.Error())
		}

		var repos []struct {
			FullName    string `json:"full_name"`
			Description string `json:"description"`
			Permissions struct {
				Admin bool `json:"admin"`
				Push  bool `json:"push"`
				Pull  bool `json:"pull"`
			} `json:"permissions"`
		}

		page := 1
		for {
			path := fmt.Sprintf("user/repos?per_page=100&sort=updated&direction=desc&page=%d", page)
			var batch []struct {
				FullName    string `json:"full_name"`
				Description string `json:"description"`
				Permissions struct {
					Admin bool `json:"admin"`
					Push  bool `json:"push"`
					Pull  bool `json:"pull"`
				} `json:"permissions"`
			}
			if err := client.Get(path, &batch); err != nil {
				return carapace.ActionMessage(err.Error())
			}
			if len(batch) == 0 {
				break
			}
			repos = append(repos, batch...)
			if len(batch) < 100 {
				break
			}
			page++
		}

		var vals []string
		for _, r := range repos {
			if r.Permissions.Push || r.Permissions.Admin {
				vals = append(vals, r.FullName, strings.TrimSpace(r.Description))
			}
		}

		return carapace.ActionValuesDescribed(vals...).Tag("repositories")
	}).Cache(24 * time.Hour)
}
