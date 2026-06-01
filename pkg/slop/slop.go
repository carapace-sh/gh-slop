package slop

import (
	"fmt"

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

type Issue struct {
	Number    int
	Author    string
	Title     string
	CreatedAt string
	State     string
}

func (i Issue) Ref(repo string) string {
	return repo + "#" + fmt.Sprint(i.Number)
}

type IssueWithRepo struct {
	Issue Issue
	Repo  string // "owner/name" for display prefix
}

func FindIssuesByAuthor(repos []repository.Repository, author string) ([]IssueWithRepo, error) {
	type repoResult struct {
		repo   string
		issues []Issue
	}

	results, err := parallelMap(repos, 5, func(r repository.Repository) (repoResult, error) {
		nodes, err := api.FetchIssuesByAuthor(r.Owner, r.Name, author)
		if err != nil {
			return repoResult{repo: r.Owner + "/" + r.Name}, fmt.Errorf("%s: %w", r.Owner+"/"+r.Name, err)
		}
		issues := make([]Issue, 0, len(nodes))
		for _, node := range nodes {
			issues = append(issues, Issue{
				Number:    node.Number,
				Author:    node.Author.Login,
				Title:     node.Title,
				CreatedAt: node.CreatedAt,
				State:     node.State,
			})
		}
		return repoResult{repo: r.Owner + "/" + r.Name, issues: issues}, nil
	})
	if err != nil {
		return nil, err
	}

	var all []IssueWithRepo
	for _, res := range results {
		for _, issue := range res.issues {
			all = append(all, IssueWithRepo{Issue: issue, Repo: res.repo})
		}
	}
	return all, nil
}

// FindPRsByAuthor finds all open PRs authored by the given user across the given repos.
func FindPRsByAuthor(repos []repository.Repository, author string) ([]PRWithRepo, error) {
	type repoResult struct {
		repo string
		prs  []PullRequest
		err  error
	}

	results, err := parallelMap(repos, 5, func(r repository.Repository) (repoResult, error) {
		nodes, err := api.FetchPullRequestsByAuthor(r.Owner, r.Name, author)
		if err != nil {
			return repoResult{repo: r.Owner + "/" + r.Name}, fmt.Errorf("%s: %w", r.Owner+"/"+r.Name, err)
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
		return repoResult{repo: r.Owner + "/" + r.Name, prs: prs}, nil
	})
	if err != nil {
		return nil, err
	}

	var all []PRWithRepo
	for _, res := range results {
		for _, pr := range res.prs {
			all = append(all, PRWithRepo{PullRequest: pr, Repo: res.repo})
		}
	}
	return all, nil
}

func ListNewContributors(repos []repository.Repository, minContributions int) ([]PRWithRepo, error) {
	type repoPRs struct {
		repo string
		prs  []PullRequest
	}

	results, err := parallelMap(repos, 5, func(r repository.Repository) (repoPRs, error) {
		prs, err := listNewContributors(r, minContributions)
		return repoPRs{repo: r.Owner + "/" + r.Name, prs: prs}, err
	})
	if err != nil {
		return nil, err
	}

	multiRepo := len(repos) > 1
	var allPRs []PRWithRepo
	for _, res := range results {
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

func listNewContributors(r repository.Repository, minContributions int) ([]PullRequest, error) {
	nodes, err := api.FetchOpenPullRequests(r.Owner, r.Name)
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

	return filterNewContributors(r, pullRequests, minContributions)
}

func filterNewContributors(r repository.Repository, pullRequests []PullRequest, minContributions int) ([]PullRequest, error) {
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

	type countResult struct {
		author string
		count  int
	}

	counts, err := parallelMap(authors, 5, func(author string) (countResult, error) {
		searchQuery := fmt.Sprintf("repo:%s/%s is:pr is:closed author:%s", r.Owner, r.Name, author)
		count, err := api.FetchMergedPRCount(searchQuery)
		return countResult{author: author, count: count}, err
	})
	if err != nil {
		return nil, err
	}

	authorCounts := map[string]int{}
	for _, res := range counts {
		authorCounts[res.author] = res.count
	}

	var newContributors []PullRequest
	for _, pr := range pullRequests {
		if authorCounts[pr.Author] < minContributions {
			newContributors = append(newContributors, pr)
		}
	}
	return newContributors, nil
}
