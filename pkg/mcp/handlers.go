package mcp

import (
	"encoding/json"
	"strings"

	"github.com/rsteube/gh-slop/pkg/slop"
)

func listReposHandler(params json.RawMessage) (string, bool) {
	repos, err := slop.AccessibleRepos()
	if err != nil {
		return err.Error(), true
	}

	var b strings.Builder
	for _, r := range repos {
		b.WriteString(r.Owner)
		b.WriteByte('/')
		b.WriteString(r.Name)
		b.WriteByte('\n')
	}
	return b.String(), false
}

func listSloppersHandler(params json.RawMessage) (string, bool) {
	var args struct {
		Repositories     []string `json:"repositories"`
		MinContributions int      `json:"min_contributions"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return err.Error(), true
	}

	repos, err := slop.ResolveRepos(args.Repositories)
	if err != nil {
		return err.Error(), true
	}

	minContrib := args.MinContributions
	if minContrib == 0 {
		minContrib = 1
	}

	prs, err := slop.ListNewContributors(repos, minContrib)
	if err != nil {
		return err.Error(), true
	}
	return formatPRs(prs), false
}

func profileSloppersHandler(params json.RawMessage) (string, bool) {
	var args struct {
		Sloppers []string `json:"sloppers"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return err.Error(), true
	}
	if len(args.Sloppers) == 0 {
		return "sloppers is required", true
	}

	profiles, err := slop.FetchUserProfiles(args.Sloppers)
	if err != nil {
		return err.Error(), true
	}
	return formatProfiles(profiles), false
}

func viewPRsHandler(params json.RawMessage) (string, bool) {
	var args struct {
		PRs []string `json:"prs"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return err.Error(), true
	}
	if len(args.PRs) == 0 {
		return "prs is required", true
	}

	details, err := slop.FetchPRDetails(args.PRs)
	if err != nil {
		return err.Error(), true
	}
	return formatPRDetails(details), false
}

func closePRsHandler(params json.RawMessage) (string, bool) {
	var args struct {
		PRs []string `json:"prs"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return err.Error(), true
	}

	if len(args.PRs) == 0 {
		return "prs is required", true
	}

	results, err := slop.ClosePRs(args.PRs)
	if err != nil {
		return err.Error(), true
	}
	return formatClosedPRs(results), false
}

func viewIssuesHandler(params json.RawMessage) (string, bool) {
	var args struct {
		Issues []string `json:"issues"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return err.Error(), true
	}
	if len(args.Issues) == 0 {
		return "issues is required", true
	}

	details, err := slop.FetchIssueDetails(args.Issues)
	if err != nil {
		return err.Error(), true
	}
	return formatIssueDetails(details), false
}

func listIssuesHandler(params json.RawMessage) (string, bool) {
	var args struct {
		Repositories []string `json:"repositories"`
		Author       string   `json:"author"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return err.Error(), true
	}
	if args.Author == "" {
		return "author is required", true
	}

	repos, err := slop.ResolveRepos(args.Repositories)
	if err != nil {
		return err.Error(), true
	}

	issues, err := slop.FindIssuesByAuthor(repos, args.Author)
	if err != nil {
		return err.Error(), true
	}
	return formatIssues(issues), false
}
