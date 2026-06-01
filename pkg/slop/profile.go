package slop

import (
	"fmt"
	"sync"
	"time"

	"github.com/rsteube/gh-slop/pkg/slop/api"
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

			resp, err := api.FetchUserProfile(login)
			if err != nil {
				results <- result{err: fmt.Errorf("%s: %w", login, err)}
				return
			}

			profile, err := toUserProfile(login, resp)
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

func toUserProfile(login string, resp api.UserProfileResponse) (UserProfile, error) {
	createdAt, err := time.Parse(time.RFC3339, resp.User.CreatedAt)
	if err != nil {
		createdAt = time.Time{}
	}

	prs := make([]UserProfilePR, 0, len(resp.User.PullRequests.Nodes))
	repoSet := map[string]bool{}
	var merged, open, closed int
	for _, node := range resp.User.PullRequests.Nodes {
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
		TotalCommits:       resp.User.ContributionsCollection.TotalCommitContributions,
		TotalPRs:           resp.User.PullRequests.TotalCount,
		MergedPRs:          merged,
		OpenPRs:            open,
		ClosedPRs:          closed,
		PRs:                prs,
		TotalReposTargeted: len(repoSet),
	}, nil
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}
