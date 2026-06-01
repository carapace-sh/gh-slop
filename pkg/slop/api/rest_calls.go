package api

import (
	"fmt"
	"strings"
)

func FetchAccessibleRepos() ([]UserRepoResponse, error) {
	client, err := RESTClient()
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
	client, err := RESTClient()
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
