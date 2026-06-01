package slop

import (
	"fmt"
	"sync"

	"github.com/rsteube/gh-slop/pkg/slop/api"
	"github.com/cli/go-gh/v2/pkg/repository"
)

type PullRequest struct {
	Number    int
	Author    string
	Title     string
	CreatedAt string
}

// Ref returns the PR reference in "OWNER/REPO#NUMBER" format.
func (pr PullRequest) Ref(repo string) string {
	return repo + "#" + fmt.Sprint(pr.Number)
}

type PRWithRepo struct {
	PullRequest PullRequest
	Repo        string // "owner/name" for display prefix
}

// FindPRsByAuthor finds all open PRs authored by the given user across the given repos.
func FindPRsByAuthor(repos []repository.Repository, author string) ([]PRWithRepo, error) {
	type result struct {
		repo  string
		prs   []PullRequest
		err   error
	}

	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup
	results := make(chan result, len(repos))

	for _, r := range repos {
		wg.Add(1)
		go func(r repository.Repository) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			client, err := api.NewGraphQLClient(r.Host)
			if err != nil {
				results <- result{repo: r.Owner + "/" + r.Name, err: fmt.Errorf("failed to create graphql client: %w", err)}
				return
			}

			nodes, err := api.FetchPullRequestsByAuthor(client, r.Owner, r.Name, author)
			if err != nil {
				results <- result{repo: r.Owner + "/" + r.Name, err: err}
				return
			}

			prs := make([]PullRequest, 0, len(nodes))
			for _, node := range nodes {
				prs = append(prs, PullRequest{
					Number:    node.Number,
					Author:    node.Author.Login,
					Title:     node.Title,
					CreatedAt: node.CreatedAt,
				})
			}
			results <- result{repo: r.Owner + "/" + r.Name, prs: prs}
		}(r)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var all []PRWithRepo
	for res := range results {
		if res.err != nil {
			return nil, fmt.Errorf("%s: %w", res.repo, res.err)
		}
		for _, pr := range res.prs {
			all = append(all, PRWithRepo{PullRequest: pr, Repo: res.repo})
		}
	}
	return all, nil
}

func ListNewContributors(repos []repository.Repository, minContributions int) ([]PRWithRepo, error) {
	multiRepo := len(repos) > 1

	type result struct {
		repo string
		prs  []PullRequest
		err  error
	}

	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup
	results := make(chan result, len(repos))

	for _, r := range repos {
		wg.Add(1)
		go func(r repository.Repository) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			client, err := api.NewGraphQLClient(r.Host)
			if err != nil {
				results <- result{repo: r.Owner + "/" + r.Name, err: fmt.Errorf("failed to create graphql client: %w", err)}
				return
			}

			prs, err := listNewContributors(client, r, minContributions)
			results <- result{repo: r.Owner + "/" + r.Name, prs: prs, err: err}
		}(r)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var allPRs []PRWithRepo
	for res := range results {
		if res.err != nil {
			return nil, fmt.Errorf("%s: %w", res.repo, res.err)
		}
		for _, pr := range res.prs {
			repoLabel := ""
			if multiRepo {
				repoLabel = res.repo
			}
			allPRs = append(allPRs, PRWithRepo{
				PullRequest: pr,
				Repo:        repoLabel,
			})
		}
	}

	return allPRs, nil
}

func listNewContributors(client api.GraphQLDoer, r repository.Repository, minContributions int) ([]PullRequest, error) {
	nodes, err := api.FetchOpenPullRequests(client, r.Owner, r.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pull requests: %w", err)
	}

	pullRequests := make([]PullRequest, 0, len(nodes))
	for _, node := range nodes {
		pullRequests = append(pullRequests, PullRequest{
			Number:    node.Number,
			Author:    node.Author.Login,
			Title:     node.Title,
			CreatedAt: node.CreatedAt,
		})
	}

	return filterNewContributors(client, r, pullRequests, minContributions)
}

func filterNewContributors(client api.GraphQLDoer, r repository.Repository, pullRequests []PullRequest, minContributions int) ([]PullRequest, error) {
	type result struct {
		author string
		count  int
		err    error
	}

	uniqueAuthors := map[string]bool{}
	for _, pr := range pullRequests {
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

			count, err := api.FetchMergedPRCount(client, searchQuery)
			if err != nil {
				results <- result{author: author, err: err}
				return
			}

			results <- result{author: author, count: count}
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

	var newContributors []PullRequest
	for _, pr := range pullRequests {
		if counts[pr.Author] < minContributions {
			newContributors = append(newContributors, pr)
		}
	}

	return newContributors, nil
}
