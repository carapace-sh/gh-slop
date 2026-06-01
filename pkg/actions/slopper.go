package actions

import (
	"fmt"
	"sort"
	"time"

	"github.com/carapace-sh/carapace"
	"github.com/carapace-sh/carapace/pkg/cache/key"
	"github.com/carapace-sh/carapace/pkg/style"
	"github.com/rsteube/gh-slop/pkg/slop"
)

// ActionSloppers completes usernames of contributors with few prior merged PRs
//
//	slopper ([1/2] full name)
//	another ([4/8] full name)
func ActionSloppers(repos ...string) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		resolvedRepos, err := slop.ResolveRepos(repos)
		if err != nil {
			return carapace.ActionMessage(err.Error())
		}

		repoKeys := make([]string, len(resolvedRepos))
		for i, r := range resolvedRepos {
			repoKeys[i] = r.Owner + "/" + r.Name
		}

		return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
			prs, err := slop.ListNewContributors(resolvedRepos, 3) // TODO config for min contributions
			if err != nil {
				return carapace.ActionMessage(err.Error())
			}

			authorPRs := groupByAuthor(prs)
			return actionSloppersValues(authorPRs)
		}).Cache(15*time.Minute, key.String(sortedStrings(repoKeys...)...))
	})
}

type slopper struct {
	author     string
	slopCount  int
	totalCount int
}

func actionSloppersValues(authorPRs map[string][]slop.PRWithRepo) carapace.Action {
	var sloppers []slopper
	for author, prs := range authorPRs {
		sloppers = append(sloppers, slopper{
			author:     author,
			slopCount:  len(prs),
			totalCount: len(prs),
		})
	}

	sort.Slice(sloppers, func(i, j int) bool {
		return sloppers[i].author < sloppers[j].author
	})

	var args []string
	for _, s := range sloppers {
		args = append(args, s.author)
		args = append(args, fmt.Sprintf("[%d/%d]", s.slopCount, s.totalCount))

		var st string
		switch {
		case s.slopCount >= 2:
			st = style.Carapace.KeywordNegative
		case s.slopCount == 1:
			st = style.Carapace.LogLevelWarning
		default:
			st = style.Carapace.Usage
		}
		args = append(args, st)
	}

	return carapace.ActionStyledValuesDescribed(args...).Tag("sloppers")
}

func groupByAuthor(prs []slop.PRWithRepo) map[string][]slop.PRWithRepo {
	grouped := map[string][]slop.PRWithRepo{}
	for _, pr := range prs {
		grouped[pr.PullRequest.Author] = append(grouped[pr.PullRequest.Author], pr)
	}
	return grouped
}

func sortedStrings(s ...string) []string {
	sort.Strings(s)
	return s
}
