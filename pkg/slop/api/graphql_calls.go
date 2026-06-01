package api

import (
	"fmt"
	"strings"
)

func FetchOpenPullRequests(owner, name string) ([]PullRequestNode, error) {
	var results []PullRequestNode
	vars := map[string]any{
		"owner": owner,
		"name":  name,
	}

	hasNext := true
	var cursor *string

	client, err := graphQLClient()
	if err != nil {
		return nil, err
	}

	for hasNext {
		vars["cursor"] = cursor
		var resp PullRequestsResponse
		if err := client.Do(QueryOpenPullRequests, vars, &resp); err != nil {
			return nil, err
		}

		for _, edge := range resp.Repository.PullRequests.Edges {
			results = append(results, edge.Node)
		}

		hasNext = resp.Repository.PullRequests.PageInfo.HasNextPage
		cursor = resp.Repository.PullRequests.PageInfo.EndCursor
	}

	return results, nil
}

func FetchPullRequestsByAuthor(owner, name, author string) ([]PullRequestNode, error) {
	var results []PullRequestNode
	searchQuery := fmt.Sprintf("repo:%s/%s is:pr is:open author:%s", owner, name, author)
	vars := map[string]any{
		"query": searchQuery,
	}

	hasNext := true
	var cursor *string

	client, err := graphQLClient()
	if err != nil {
		return nil, err
	}

	for hasNext {
		vars["cursor"] = cursor
		var resp SearchPullRequestsResponse
		if err := client.Do(QuerySearchPullRequests, vars, &resp); err != nil {
			return nil, err
		}

		for _, edge := range resp.Search.Edges {
			results = append(results, edge.Node)
		}

		hasNext = resp.Search.PageInfo.HasNextPage
		cursor = resp.Search.PageInfo.EndCursor
	}

	return results, nil
}

func FetchIssuesByAuthor(owner, name, author string) ([]IssueNode, error) {
	var results []IssueNode
	searchQuery := fmt.Sprintf("repo:%s/%s is:issue author:%s", owner, name, author)
	vars := map[string]any{
		"query": searchQuery,
	}

	hasNext := true
	var cursor *string

	client, err := graphQLClient()
	if err != nil {
		return nil, err
	}

	for hasNext {
		vars["cursor"] = cursor
		var resp SearchIssuesResponse
		if err := client.Do(QuerySearchIssues, vars, &resp); err != nil {
			return nil, err
		}

		for _, edge := range resp.Search.Edges {
			results = append(results, edge.Node)
		}

		hasNext = resp.Search.PageInfo.HasNextPage
		cursor = resp.Search.PageInfo.EndCursor
	}

	return results, nil
}

func FetchMergedPRCount(searchQuery string) (int, error) {
	client, err := graphQLClient()
	if err != nil {
		return 0, err
	}

	vars := map[string]any{
		"query": searchQuery,
	}
	var resp MergedPRCountResponse
	if err := client.Do(QueryMergedPRCount, vars, &resp); err != nil {
		return 0, err
	}
	return resp.Search.IssueCount, nil
}

func FetchUserProfile(login string) (UserProfileResponse, error) {
	client, err := graphQLClient()
	if err != nil {
		return UserProfileResponse{}, err
	}

	vars := map[string]any{
		"login": login,
	}
	var resp UserProfileResponse
	if err := client.Do(QueryUserProfile, vars, &resp); err != nil {
		return UserProfileResponse{}, fmt.Errorf("failed to fetch user profile: %w", err)
	}
	return resp, nil
}

func FetchPRDetailsForRepo(owner, name string, numbers []int) (map[string]any, error) {
	client, err := graphQLClient()
	if err != nil {
		return nil, err
	}

	vars := map[string]any{
		"owner": owner,
		"name":  name,
	}

	var aliases []string
	for i, num := range numbers {
		alias := fmt.Sprintf("pr%d", i)
		vars[alias] = num
		aliases = append(aliases, fmt.Sprintf(
			`%s: pullRequest(number: $%s) { number title body author { login } createdAt url }`,
			alias, alias,
		))
	}

	var buf strings.Builder
	for i := range numbers {
		if i > 0 {
			buf.WriteString(", ")
		}
		fmt.Fprintf(&buf, "$pr%d: Int!", i)
	}

	query := fmt.Sprintf(`query($owner: String!, $name: String!, %s) {
  repository(owner: $owner, name: $name) {
    %s
  }
}`,
		buf.String(),
		strings.Join(aliases, "\n    "),
	)

	var response map[string]any
	if err := client.Do(query, vars, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch PR details: %w", err)
	}

	return response, nil
}

type PullRequestNode struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	CreatedAt string `json:"createdAt"`
	Author    struct {
		Login string `json:"login"`
	} `json:"author"`
	Repository struct {
		NameWithOwner string `json:"nameWithOwner"`
	} `json:"repository"`
}
