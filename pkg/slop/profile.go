package slop

import (
	"fmt"
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
	profiles, err := parallelMap(logins, 5, func(login string) (UserProfile, error) {
		resp, err := api.FetchUserProfile(login)
		if err != nil {
			return UserProfile{}, fmt.Errorf("%s: %w", login, err)
		}
		profile, err := toUserProfile(login, resp)
		if err != nil {
			return UserProfile{}, fmt.Errorf("%s: %w", login, err)
		}
		return profile, nil
	})
	if err != nil {
		return nil, err
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
