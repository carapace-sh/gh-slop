package api

import (
	"fmt"
	"strings"
)

func FetchAccessibleRepos() ([]UserRepoResponse, error) {
	client, err := restClient()
	if err != nil {
		return nil, err
	}

	var allRepos []UserRepoResponse

	page := 1
	for {
		path := fmt.Sprintf("user/repos?per_page=100&sort=updated&direction=desc&page=%d", page)
		var batch []UserRepoResponse
		if err := client.Get(path, &batch); err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		allRepos = append(allRepos, batch...)
		if len(batch) < 100 {
			break
		}
		page++
	}

	return allRepos, nil
}

func ClosePR(repo string, number int) (string, error) {
	client, err := restClient()
	if err != nil {
		return "", err
	}

	path := fmt.Sprintf("repos/%s/pulls/%d", repo, number)
	body := strings.NewReader(`{"state":"closed"}`)
	var resp map[string]any
	if err := client.Patch(path, body, &resp); err != nil {
		return "", err
	}

	state, _ := resp["state"].(string)
	if state == "" {
		state = "closed"
	}
	return state, nil
}

type IssueResponse struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	State     string `json:"state"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	URL       string `json:"html_url"`
	Author    struct {
		Login string `json:"login"`
	} `json:"user"`
}

func FetchIssue(repo string, number int) (IssueResponse, error) {
	client, err := restClient()
	if err != nil {
		return IssueResponse{}, err
	}

	path := fmt.Sprintf("repos/%s/issues/%d", repo, number)
	var resp IssueResponse
	if err := client.Get(path, &resp); err != nil {
		return IssueResponse{}, err
	}
	return resp, nil
}
