package mcp

import (
	"fmt"
	"strings"

	"github.com/rsteube/gh-slop/pkg/slop"
)

func formatPRs(prs []slop.PRWithRepo) string {
	if len(prs) == 0 {
		return "No new contributors found."
	}
	var b strings.Builder
	for _, pr := range prs {
		fmt.Fprintf(&b, "#%d: %s (@%s)\n", pr.PullRequest.Number, pr.PullRequest.Title, pr.PullRequest.Author)
	}
	return b.String()
}

func formatIssues(issues []slop.IssueWithRepo) string {
	if len(issues) == 0 {
		return "No issues found."
	}
	var b strings.Builder
	for _, issue := range issues {
		fmt.Fprintf(&b, "%s#%d: %s [%s] (@%s)\n", issue.Repo, issue.Issue.Number, issue.Issue.Title, issue.Issue.State, issue.Issue.Author)
	}
	return b.String()
}

func formatProfiles(profiles []slop.UserProfile) string {
	var b strings.Builder
	for i, p := range profiles {
		if i > 0 {
			b.WriteByte('\n')
		}
		mergeRate := 0
		if p.TotalPRs > 0 {
			mergeRate = p.MergedPRs * 100 / p.TotalPRs
		}
		fmt.Fprintf(&b, "## @%s\n", p.Login)
		fmt.Fprintf(&b, "Account created: %s\n", p.CreatedAt.Format("2006-01-02"))
		fmt.Fprintf(&b, "Total commits: %d\n", p.TotalCommits)
		fmt.Fprintf(&b, "Total PRs: %d (merged: %d, open: %d, closed: %d) — %d%% merge rate\n", p.TotalPRs, p.MergedPRs, p.OpenPRs, p.ClosedPRs, mergeRate)
		fmt.Fprintf(&b, "Repos targeted: %d\n", p.TotalReposTargeted)
		if len(p.PRs) > 0 {
			b.WriteString("Recent PRs:\n")
			for _, pr := range p.PRs {
				fmt.Fprintf(&b, "  - [%s] %s (%s)\n", pr.State, pr.Title, pr.Repo)
			}
		}
	}
	return b.String()
}

func formatPRDetails(details []slop.PRDetail) string {
	var b strings.Builder
	for i, d := range details {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "## %s#%d\n", d.Repo, d.Number)
		fmt.Fprintf(&b, "Title: %s\n", d.Title)
		fmt.Fprintf(&b, "Author: @%s\n", d.Author)
		fmt.Fprintf(&b, "Created: %s\n", d.CreatedAt)
		fmt.Fprintf(&b, "URL: %s\n", d.URL)
		if d.Body != "" {
			b.WriteString("---\n")
			b.WriteString(htmlEscape(d.Body))
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func formatClosedPRs(results []slop.ClosedPR) string {
	var b strings.Builder
	for i, r := range results {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%s#%d: %s", r.Repo, r.Number, r.State)
	}
	return b.String()
}

func formatIssueDetails(details []slop.IssueDetail) string {
	var b strings.Builder
	for i, d := range details {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "## %s#%d\n", d.Repo, d.Number)
		fmt.Fprintf(&b, "Title: %s\n", d.Title)
		fmt.Fprintf(&b, "Author: @%s\n", d.Author)
		fmt.Fprintf(&b, "State: %s\n", d.State)
		fmt.Fprintf(&b, "Created: %s\n", d.CreatedAt)
		fmt.Fprintf(&b, "Updated: %s\n", d.UpdatedAt)
		fmt.Fprintf(&b, "URL: %s\n", d.URL)
		if d.Body != "" {
			b.WriteString("---\n")
			b.WriteString(htmlEscape(d.Body))
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func htmlEscape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '<':
			b.WriteString("<")
		case '>':
			b.WriteString(">")
		case '&':
			b.WriteString("&")
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
