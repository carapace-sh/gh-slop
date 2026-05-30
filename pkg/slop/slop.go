package slop

import (
	"fmt"
	"sync"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
)

type PullRequest struct {
	Number    int
	Author    string
	Title     string
	CreatedAt string
}

func ListNewContributors(r repository.Repository, minContributions int) ([]PullRequest, error) {
	client, err := api.NewGraphQLClient(api.ClientOptions{Host: r.Host})
	if err != nil {
		return nil, fmt.Errorf("failed to create graphql client: %w", err)
	}

	pullRequests, err := fetchPullRequests(client, r)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pull requests: %w", err)
	}

	contributionCounts, err := fetchContributionCounts(client, r, pullRequests)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch contribution counts: %w", err)
	}

	var newContributors []PullRequest
	for _, pr := range pullRequests {
		if contributionCounts[pr.Author] < minContributions {
			newContributors = append(newContributors, pr)
		}
	}

	return newContributors, nil
}

func fetchContributionCounts(client graphqlDoer, r repository.Repository, openPRs []PullRequest) (map[string]int, error) {
	type result struct {
		author string
		count  int
		err    error
	}

	uniqueAuthors := map[string]bool{}
	for _, pr := range openPRs {
		if pr.Author != "" {
			uniqueAuthors[pr.Author] = true
		}
	}

	authors := make([]string, 0, len(uniqueAuthors))
	for a := range uniqueAuthors {
		authors = append(authors, a)
	}

	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup
	results := make(chan result, len(authors))

	for _, author := range authors {
		wg.Add(1)
		go func(author string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			searchQuery := fmt.Sprintf("repo:%s/%s is:pr is:closed author:%s", r.Owner, r.Name, author)

			vars := map[string]any{
				"query": searchQuery,
			}

			var response struct {
				Search struct {
					IssueCount int `json:"issueCount"`
				} `json:"search"`
			}

			query := `
				query($query: String!) {
					search(query: $query, type: ISSUE, first: 1) {
						issueCount
					}
				}`

			if err := client.Do(query, vars, &response); err != nil {
				results <- result{author: author, err: err}
				return
			}

			results <- result{author: author, count: response.Search.IssueCount}
		}(author)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	counts := map[string]int{}
	for res := range results {
		if res.err != nil {
			return nil, res.err
		}
		counts[res.author] = res.count
	}

	return counts, nil
}

func fetchPullRequests(client graphqlDoer, r repository.Repository) ([]PullRequest, error) {
	var results []PullRequest
	vars := map[string]any{
		"owner": r.Owner,
		"name":  r.Name,
	}

	hasNext := true
	var cursor *string

	for hasNext {
		vars["cursor"] = cursor
		var response struct {
			Repository struct {
				PullRequests struct {
					PageInfo struct {
						HasNextPage bool    `json:"hasNextPage"`
						EndCursor   *string `json:"endCursor"`
					} `json:"pageInfo"`
					Edges []struct {
						Node struct {
							Number    int    `json:"number"`
							Title     string `json:"title"`
							CreatedAt string `json:"createdAt"`
							Author struct {
								Login string `json:"login"`
							} `json:"author"`
						} `json:"node"`
					} `json:"edges"`
				} `json:"pullRequests"`
			} `json:"repository"`
		}

		query := `
			query($owner: String!, $name: String!, $cursor: String) {
				repository(owner: $owner, name: $name) {
					pullRequests(first: 100, after: $cursor, states: [OPEN]) {
						pageInfo { hasNextPage endCursor }
						edges { node { number title createdAt author { login } } }
					}
				}
			}`

		err := client.Do(query, vars, &response)
		if err != nil {
			return nil, err
		}

		for _, edge := range response.Repository.PullRequests.Edges {
			results = append(results, PullRequest{
				Number:    edge.Node.Number,
				Author:    edge.Node.Author.Login,
				Title:     edge.Node.Title,
				CreatedAt: edge.Node.CreatedAt,
			})
		}

		hasNext = response.Repository.PullRequests.PageInfo.HasNextPage
		cursor = response.Repository.PullRequests.PageInfo.EndCursor
	}

	return results, nil
}

type graphqlDoer interface {
	Do(query string, variables map[string]any, response any) error
}
