package slop

import (
	"fmt"
	"strings"
	"sync"

	"github.com/cli/go-gh/v2/pkg/api"
)

type ClosedPR struct {
	Repo   string // "owner/repo"
	Number int
	State  string // "closed" on success, or error message
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

			client, err := api.DefaultRESTClient()
			if err != nil {
				ch <- closeResult{index: i, err: fmt.Errorf("%s: failed to create REST client: %w", r.ref, err)}
				return
			}

			path := fmt.Sprintf("repos/%s/pulls/%d", r.repo, r.number)
			body := strings.NewReader(`{"state":"closed"}`)
			var resp map[string]any
			if err := client.Patch(path, body, &resp); err != nil {
				ch <- closeResult{index: i, closed: ClosedPR{Repo: r.repo, Number: r.number, State: fmt.Sprintf("error: %v", err)}}
				return
			}

			state, _ := resp["state"].(string)
			if state == "" {
				state = "closed"
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
