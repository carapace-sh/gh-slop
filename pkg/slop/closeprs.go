package slop

import (
	"fmt"
	"sync"

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

	type closeResult struct {
		index int
		closed ClosedPR
		err    error
	}

	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup
	ch := make(chan closeResult, len(refs))

	for i, r := range refs {
		wg.Add(1)
		go func(i int, r parsedRef) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			state, err := api.ClosePR(r.repo, r.number)
			if err != nil {
				ch <- closeResult{index: i, closed: ClosedPR{Repo: r.repo, Number: r.number, State: fmt.Sprintf("error: %v", err)}}
				return
			}

			ch <- closeResult{index: i, closed: ClosedPR{Repo: r.repo, Number: r.number, State: state}}
		}(i, r)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	results := make([]ClosedPR, len(refs))
	for res := range ch {
		if res.err != nil {
			return nil, res.err
		}
		results[res.index] = res.closed
	}

	return results, nil
}
