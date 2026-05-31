package render

import (
	"fmt"
	"sort"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/list"
	"github.com/rsteube/gh-slop/pkg/slop"
)

type PRWithRepo = slop.PRWithRepo

var (
	authorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#4a6892", Dark: "#87a7d9"}).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#6b6f76", Dark: "#9aa0aa"})

	whiteStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#333333", Dark: "#eeeeee"})

	yellowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#b58b00", Dark: "#ffd666"})

	redStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#cc3333", Dark: "#ff6666"})
)

func Render(prs []slop.PRWithRepo) string {
	if len(prs) == 0 {
		return "No first-time contributors with open pull requests."
	}

	grouped := groupByAuthor(prs)
	authors := sortedKeys(grouped)

	var sections []string
	for _, author := range authors {
		prList := grouped[author]
		sortByCreatedAt(prList)

		parsedTimes := parseTimes(prList)
		clusters := clusterByTime(parsedTimes, time.Hour)

		var items []string
		for j, pr := range prList {
			entry := prLabel(pr) + "  " + formatTime(pr.PullRequest.CreatedAt) + "  " + pr.PullRequest.Title

			if cluster, ok := clusters[j]; ok {
				switch cluster.Position {
				case 0:
					entry = whiteStyle.Render(entry)
				case 1:
					entry = yellowStyle.Render(entry)
				default:
					entry = redStyle.Render(entry)
				}
			} else {
				entry = dimStyle.Render(entry)
			}

			items = append(items, entry)
		}

		anyItems := make([]any, len(items))
		for i, item := range items {
			anyItems[i] = item
		}

		l := list.New(anyItems...).
			Enumerator(list.Dash)

		sections = append(sections, authorStyle.Render("@"+author)+"\n"+l.String())
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func prLabel(pr slop.PRWithRepo) string {
	if pr.Repo != "" {
		return fmt.Sprintf("%s#%d", pr.Repo, pr.PullRequest.Number)
	}
	return fmt.Sprintf("#%d", pr.PullRequest.Number)
}

func groupByAuthor(prs []slop.PRWithRepo) map[string][]slop.PRWithRepo {
	grouped := map[string][]slop.PRWithRepo{}
	for _, pr := range prs {
		grouped[pr.PullRequest.Author] = append(grouped[pr.PullRequest.Author], pr)
	}
	return grouped
}

func sortedKeys(m map[string][]slop.PRWithRepo) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortByCreatedAt(prs []slop.PRWithRepo) {
	sort.Slice(prs, func(i, j int) bool {
		return prs[i].PullRequest.CreatedAt < prs[j].PullRequest.CreatedAt
	})
}

func parseTimes(prs []slop.PRWithRepo) []time.Time {
	times := make([]time.Time, len(prs))
	for i, pr := range prs {
		t, err := time.Parse(time.RFC3339, pr.PullRequest.CreatedAt)
		if err != nil {
			continue
		}
		times[i] = t
	}
	return times
}

type clusterStep struct {
	Size     int
	Position int
}

func clusterByTime(times []time.Time, threshold time.Duration) map[int]clusterStep {
	if len(times) == 0 {
		return nil
	}

	groupIDs := make([]int, len(times))
	currentGroup := 0
	groupIDs[0] = 0

	for i := 1; i < len(times); i++ {
		if times[i].Sub(times[i-1]) <= threshold {
			groupIDs[i] = currentGroup
		} else {
			currentGroup++
			groupIDs[i] = currentGroup
		}
	}

	groupSizes := map[int]int{}
	for _, gid := range groupIDs {
		groupSizes[gid]++
	}

	positions := map[int]int{}
	result := map[int]clusterStep{}

	for i, gid := range groupIDs {
		pos := positions[gid]
		positions[gid] = pos + 1

		if groupSizes[gid] >= 2 {
			result[i] = clusterStep{
				Size:     groupSizes[gid],
				Position: pos,
			}
		}
	}

	return result
}

func formatTime(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return t.Format("2006-01-02 15:04")
}
