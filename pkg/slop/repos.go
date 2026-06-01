package slop

import (
	"fmt"

	"github.com/rsteube/gh-slop/pkg/slop/api"
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
	client, err := api.NewDefaultRESTClient()
	if err != nil {
		return nil, err
	}

	allRepos, err := api.FetchAccessibleRepos(client)
	if err != nil {
		return nil, err
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
