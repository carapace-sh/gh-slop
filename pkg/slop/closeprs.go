package slop

import (
	"fmt"

	"github.com/rsteube/gh-slop/pkg/slop/api"
)

type ClosedPR struct {
	Repo   string // "owner/repo"
	Number int
	State  string // "closed" on success, or error message
}

// Ref returns the PR reference in "OWNER/REPO#NUMBER" format.
func (r ClosedPR) Ref() string {
	return r.Repo + "#" + fmt.Sprint(r.Number)
}

// ClosePRs closes the given pull requests via the GitHub REST API.
// PRs are specified in "OWNER/REPO#NUMBER" format.
func ClosePRs(prRefs []string) ([]ClosedPR, error) {
	type parsedRef struct {
		ref    string
		repo   string // "owner/repo"
		host   string
		number int
	}

	var refs []parsedRef
	for _, ref := range prRefs {
		repo, number, err := ParsePRRef(ref)
		if err != nil {
			return nil, err
		}
		refs = append(refs, parsedRef{
			ref:    ref,
			repo:   repo.Owner + "/" + repo.Name,
			host:   repo.Host,
			number: number,
		})
	}

	results, err := parallelMap(refs, 5, func(r parsedRef) (ClosedPR, error) {
		state, err := api.ClosePR(r.repo, r.number)
		if err != nil {
			return ClosedPR{Repo: r.repo, Number: r.number, State: fmt.Sprintf("error: %v", err)}, nil
		}
		return ClosedPR{Repo: r.repo, Number: r.number, State: state}, nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}
