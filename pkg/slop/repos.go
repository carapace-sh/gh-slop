package slop

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
)

func Repos(repoFilters []string) ([]repository.Repository, error) {
	client, err := api.DefaultRESTClient()
	if err != nil {
		return nil, err
	}

	var allRepos []struct {
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
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		allRepos = append(allRepos, batch...)
		if len(batch) < 100 {
			break
		}
		page++
	}

	// If no filters provided, return all accessible repos
	if len(repoFilters) == 0 {
		result := make([]repository.Repository, 0, len(allRepos))
		for _, r := range allRepos {
			if r.Permissions.Push || r.Permissions.Admin {
				parsed, err := repository.Parse(r.FullName)
				if err != nil {
					return nil, fmt.Errorf("failed to parse repo %q: %w", r.FullName, err)
				}
				result = append(result, parsed)
			}
		}
		return result, nil
	}

	// Filter to only the requested repos
	result := make([]repository.Repository, 0, len(repoFilters))
	for _, filter := range repoFilters {
		parsed, err := repository.Parse(filter)
		if err != nil {
			return nil, fmt.Errorf("failed to parse repo %q: %w", filter, err)
		}
		// Check if user has access
		hasAccess := false
		for _, r := range allRepos {
			if r.FullName == filter && (r.Permissions.Push || r.Permissions.Admin) {
				hasAccess = true
				break
			}
		}
		if hasAccess {
			result = append(result, parsed)
		}
	}
	return result, nil
}
