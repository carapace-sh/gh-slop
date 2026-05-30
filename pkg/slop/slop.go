package slop

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
)

type PullRequest struct {
	Number int
	Author string
	Title  string
}

func ListNewContributors(r repository.Repository, minContributions int) ([]PullRequest, error) {
	client, err := api.NewGraphQLClient(api.ClientOptions{Host: r.Host})
	if err != nil {
		return nil, fmt.Errorf("failed to create graphql client: %w", err)
	}

	contributionCounts, err := fetchExistingContributors(client, r)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch existing contributors: %w", err)
	}

	pullRequests, err := fetchPullRequests(client, r)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pull requests: %w", err)
	}

	var newContributors []PullRequest
	for _, pr := range pullRequests {
		if contributionCounts[pr.Author] < minContributions {
			newContributors = append(newContributors, pr)
		}
	}

	return newContributors, nil
}

func fetchExistingContributors(client graphqlDoer, r repository.Repository) (map[string]int, error) {
	contributors := map[string]int{}
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
					pullRequests(first: 100, after: $cursor, states: [MERGED, CLOSED]) {
						pageInfo { hasNextPage endCursor }
						edges { node { author { login } } }
					}
				}
			}`

		err := client.Do(query, vars, &response)
		if err != nil {
			return nil, err
		}

		for _, edge := range response.Repository.PullRequests.Edges {
			login := edge.Node.Author.Login
			if login != "" {
				contributors[login]++
			}
		}

		hasNext = response.Repository.PullRequests.PageInfo.HasNextPage
		cursor = response.Repository.PullRequests.PageInfo.EndCursor
	}

	return contributors, nil
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
							Number int    `json:"number"`
							Title  string `json:"title"`
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
						edges { node { number title author { login } } }
					}
				}
			}`

		err := client.Do(query, vars, &response)
		if err != nil {
			return nil, err
		}

		for _, edge := range response.Repository.PullRequests.Edges {
			results = append(results, PullRequest{
				Number: edge.Node.Number,
				Author: edge.Node.Author.Login,
				Title:  edge.Node.Title,
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
