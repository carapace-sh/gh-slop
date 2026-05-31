package actions

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
)

func ResolveRepos(repos []string) ([]repository.Repository, error) {
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
