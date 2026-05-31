package slop

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
)

// ResolveRepos resolves repository filters to concrete repositories.
// When no filters are provided, it uses the current repository.
func ResolveRepos(repoFilters []string) ([]repository.Repository, error) {
	if len(repoFilters) == 0 {
		repo, err := repository.Current()
		if err != nil {
			return nil, fmt.Errorf("no repository specified and not in a git repo: %w", err)
		}
		return []repository.Repository{repo}, nil
	}

	result := make([]repository.Repository, 0, len(repoFilters))
	for _, r := range repoFilters {
		parsed, err := repository.Parse(r)
		if err != nil {
			return nil, fmt.Errorf("failed to parse repo %q: %w", r, err)
		}
		result = append(result, parsed)
	}
	return result, nil
}

// AccessibleRepos fetches all repositories the current user has push or admin access to.
func AccessibleRepos() ([]repository.Repository, error) {
	client, err := api.DefaultRESTClient()
	if err != nil {
		return nil, err
	}

	var allRepos []struct {
		FullName    string `json:"full_name"`
		Permissions struct {
			Admin bool `json:"admin"`
			Push  bool `json:"push"`
		} `json:"permissions"`
	}

	page := 1
	for {
		path := fmt.Sprintf("user/repos?per_page=100&sort=updated&direction=desc&page=%d", page)
		var batch []struct {
			FullName    string `json:"full_name"`
			Permissions struct {
				Admin bool `json:"admin"`
				Push  bool `json:"push"`
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
