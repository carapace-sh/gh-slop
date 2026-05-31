package slop

import (
	"fmt"
	"sync"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

type UserProfile struct {
	Login              string
	CreatedAt          time.Time
	TotalCommits       int
	TotalPRs           int
	MergedPRs          int
	OpenPRs            int
	ClosedPRs          int
	PRs                []UserProfilePR
	TotalReposTargeted int
}

type UserProfilePR struct {
	Repo      string
	Title     string
	CreatedAt time.Time
	State     string
}

func FetchUserProfiles(logins []string) ([]UserProfile, error) {
	type result struct {
		profile UserProfile
		err     error
	}

	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup
	results := make(chan result, len(logins))

	for _, login := range logins {
		wg.Add(1)
		go func(login string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			client, err := api.NewGraphQLClient(api.ClientOptions{})
			if err != nil {
				results <- result{err: fmt.Errorf("%s: failed to create graphql client: %w", login, err)}
				return
			}

			profile, err := fetchUserProfile(client, login)
			if err != nil {
				results <- result{err: fmt.Errorf("%s: %w", login, err)}
				return
			}
			results <- result{profile: profile}
		}(login)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var profiles []UserProfile
	for res := range results {
		if res.err != nil {
			return nil, res.err
		}
		profiles = append(profiles, res.profile)
	}

	return profiles, nil
}

func fetchUserProfile(client graphqlDoer, login string) (UserProfile, error) {
	vars := map[string]any{
		"login": login,
	}

	var response struct {
		User struct {
			CreatedAt string `json:"createdAt"`
			ContributionsCollection struct {
				TotalCommitContributions int `json:"totalCommitContributions"`
			} `json:"contributionsCollection"`
			PullRequests struct {
				TotalCount int `json:"totalCount"`
				Nodes      []struct {
					Repository struct {
						NameWithOwner string `json:"nameWithOwner"`
					} `json:"repository"`
					Title     string `json:"title"`
					CreatedAt string `json:"createdAt"`
					State     string `json:"state"`
				} `json:"nodes"`
			} `json:"pullRequests"`
		} `json:"user"`
	}

	query := `
		query($login: String!) {
			user(login: $login) {
				createdAt
				contributionsCollection {
					totalCommitContributions
				}
				pullRequests(first: 50, orderBy: {field: CREATED_AT, direction: DESC}) {
					totalCount
					nodes {
						repository { nameWithOwner }
						title
						createdAt
						state
					}
				}
			}
		}`

	if err := client.Do(query, vars, &response); err != nil {
		return UserProfile{}, fmt.Errorf("failed to fetch user profile: %w", err)
	}

	createdAt, err := time.Parse(time.RFC3339, response.User.CreatedAt)
	if err != nil {
		createdAt = time.Time{}
	}

	prs := make([]UserProfilePR, 0, len(response.User.PullRequests.Nodes))
	repoSet := map[string]bool{}
	var merged, open, closed int
	for _, node := range response.User.PullRequests.Nodes {
		prs = append(prs, UserProfilePR{
			Repo:      node.Repository.NameWithOwner,
			Title:     node.Title,
			CreatedAt: parseTime(node.CreatedAt),
			State:     node.State,
		})
		repoSet[node.Repository.NameWithOwner] = true
		switch node.State {
		case "MERGED":
			merged++
		case "OPEN":
			open++
		case "CLOSED":
			closed++
		}
	}

	return UserProfile{
		Login:              login,
		CreatedAt:          createdAt,
		TotalCommits:       response.User.ContributionsCollection.TotalCommitContributions,
		TotalPRs:           response.User.PullRequests.TotalCount,
		MergedPRs:          merged,
		OpenPRs:            open,
		ClosedPRs:          closed,
		PRs:                prs,
		TotalReposTargeted: len(repoSet),
	}, nil
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}