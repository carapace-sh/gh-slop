package slop

import (
	"fmt"
	"sync"

	"github.com/rsteube/gh-slop/pkg/slop/api"
	"github.com/cli/go-gh/v2/pkg/repository"
)

type PRDetail struct {
	Repo      string // "owner/repo"
	Number    int
	Title     string
	Body      string
	Author    string
	CreatedAt string
	URL       string
}

// ParsePRRef parses a PR reference in "OWNER/REPO#NUMBER" format
// and returns the repository and PR number.
func ParsePRRef(ref string) (repository.Repository, int, error) {
	var owner, rest string
	for i := 0; i < len(ref); i++ {
		if ref[i] == '/' {
			owner = ref[:i]
			rest = ref[i+1:]
			break
		}
	}
	if owner == "" || rest == "" {
		return repository.Repository{}, 0, fmt.Errorf("invalid PR reference format %q, expected OWNER/REPO#NUMBER", ref)
	}

	var repo string
	var numStr string
	for i := len(rest) - 1; i >= 0; i-- {
		if rest[i] == '#' {
			repo = rest[:i]
			numStr = rest[i+1:]
			break
		}
	}
	if repo == "" || numStr == "" {
		return repository.Repository{}, 0, fmt.Errorf("invalid PR reference format %q, expected OWNER/REPO#NUMBER", ref)
	}

	var number int
	if _, err := fmt.Sscanf(numStr, "%d", &number); err != nil {
		return repository.Repository{}, 0, fmt.Errorf("invalid PR number in %q: %w", ref, err)
	}

	parsed, err := repository.Parse(owner + "/" + repo)
	if err != nil {
		return repository.Repository{}, 0, fmt.Errorf("invalid repo in %q: %w", ref, err)
	}

	return parsed, number, nil
}

// FetchPRDetails fetches title, body, createdAt, author, and URL for a list of PRs.
// PRs are specified in "OWNER/REPO#NUMBER" format.
// PRs are grouped by repo and fetched concurrently with one GraphQL request per repo.
func FetchPRDetails(prRefs []string) ([]PRDetail, error) {
	type prKey struct {
		owner string
		name  string
	}

	grouped := map[prKey][]int{}
	refIndex := map[prKey]map[int]int{}

	for i, ref := range prRefs {
		repo, number, err := ParsePRRef(ref)
		if err != nil {
			return nil, err
		}
		key := prKey{owner: repo.Owner, name: repo.Name}
		grouped[key] = append(grouped[key], number)
		if refIndex[key] == nil {
			refIndex[key] = map[int]int{}
		}
		refIndex[key][number] = i
	}

	client, err := api.NewDefaultGraphQLClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create graphql client: %w", err)
	}

	type prResult struct {
		key     prKey
		details []PRDetail
		err     error
	}

	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup
	ch := make(chan prResult, len(grouped))

	for key, numbers := range grouped {
		wg.Add(1)
		go func(key prKey, numbers []int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			response, err := api.FetchPRDetailsForRepo(client, key.owner, key.name, numbers)
			if err != nil {
				ch <- prResult{key: key, err: err}
				return
			}

			details := toPRDetails(key.owner, key.name, numbers, response)
			ch <- prResult{key: key, details: details}
		}(key, numbers)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	ordered := make([]PRDetail, len(prRefs))
	for res := range ch {
		if res.err != nil {
			return nil, fmt.Errorf("%s/%s: %w", res.key.owner, res.key.name, res.err)
		}
		for _, d := range res.details {
			key := prKey{owner: res.key.owner, name: res.key.name}
			idx := refIndex[key][d.Number]
			ordered[idx] = d
		}
	}

	var out []PRDetail
	for _, d := range ordered {
		if d.Number != 0 {
			out = append(out, d)
		}
	}
	return out, nil
}

func toPRDetails(owner, name string, numbers []int, response map[string]any) []PRDetail {
	repo, ok := response["repository"].(map[string]any)
	if !ok {
		return nil
	}

	details := make([]PRDetail, 0, len(numbers))
	for i, num := range numbers {
		alias := fmt.Sprintf("pr%d", i)
		prData, ok := repo[alias].(map[string]any)
		if !ok {
			continue
		}
		detail := PRDetail{
			Repo:      owner + "/" + name,
			Number:    num,
			Title:     strVal(prData["title"]),
			Body:      strVal(prData["body"]),
			CreatedAt: strVal(prData["createdAt"]),
			URL:       strVal(prData["url"]),
		}
		if author, ok := prData["author"].(map[string]any); ok {
			detail.Author = strVal(author["login"])
		}
		details = append(details, detail)
	}
	return details
}

func strVal(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
