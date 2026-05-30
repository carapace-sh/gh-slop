package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/carapace-sh/carapace"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/rsteube/gh-slop/pkg/slop"
	"github.com/spf13/cobra"
)

var minContributions int

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List pull requests from new contributors",
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := resolveRepo()
		if err != nil {
			return err
		}

		prs, err := slop.ListNewContributors(r, minContributions)
		if err != nil {
			return err
		}

		if len(prs) == 0 {
			fmt.Println("No first-time contributors with open pull requests.")
			return nil
		}

		grouped := groupByAuthor(prs)
		authors := sortedKeys(grouped)

		authorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Bold(true)

		headerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

		sepStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

		authorRowIndices := map[int]bool{}
		sepRowIndices := map[int]bool{}
		prNumberStyles := map[int]lipgloss.Style{}

		var rows [][]string
		for i, author := range authors {
			prList := grouped[author]
			sortByCreatedAt(prList)

			if i > 0 {
				sepRowIndices[len(rows)] = true
				rows = append(rows, []string{strings.Repeat("─", 40), "", ""})
			}
			authorRowIndices[len(rows)] = true
			rows = append(rows, []string{"@" + author, "", ""})

			parsedTimes := parseTimes(prList)
			clusters := clusterByTime(parsedTimes, time.Hour)

			for j, pr := range prList {
				prNum := fmt.Sprintf("#%d", pr.Number)
				rows = append(rows, []string{
					prNum,
					formatTime(pr.CreatedAt),
					pr.Title,
				})

					if cluster, ok := clusters[j]; ok && cluster.Highlight {
					degree := float64(cluster.Position) / float64(cluster.Size-1)
					g := int(255 * (1 - degree))
					prNumberStyles[len(rows)-1] = lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#ff%02x%02x", g, g)))
				}
			}
		}

		t := table.New().
			Headers("PR", "Created", "Title").
			Rows(rows...).
			StyleFunc(func(row, col int) lipgloss.Style {
				switch {
				case row == table.HeaderRow:
					return headerStyle
				case sepRowIndices[row]:
					return sepStyle
				case authorRowIndices[row]:
					return authorStyle
				case col == 0:
					if style, ok := prNumberStyles[row]; ok {
						return style
					}
				}
				return lipgloss.NewStyle()
			}).
			BorderLeft(false).
			BorderRight(false).
			BorderTop(false).
			BorderBottom(false).
			BorderHeader(false).
			BorderColumn(false)

		fmt.Println(t.Render())

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().IntVarP(&minContributions, "min-contributions", "m", 1, "Minimum merged PRs for a contributor to be filtered out")
	carapace.Gen(listCmd)
}

func groupByAuthor(prs []slop.PullRequest) map[string][]slop.PullRequest {
	grouped := map[string][]slop.PullRequest{}
	for _, pr := range prs {
		grouped[pr.Author] = append(grouped[pr.Author], pr)
	}
	return grouped
}

func sortedKeys(m map[string][]slop.PullRequest) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortByCreatedAt(prs []slop.PullRequest) {
	sort.Slice(prs, func(i, j int) bool {
		return prs[i].CreatedAt < prs[j].CreatedAt
	})
}

func parseTimes(prs []slop.PullRequest) []time.Time {
	times := make([]time.Time, len(prs))
	for i, pr := range prs {
		t, err := time.Parse(time.RFC3339, pr.CreatedAt)
		if err != nil {
			continue
		}
		times[i] = t
	}
	return times
}

type clusterStep struct {
	Size       int
	Position   int
	Highlight  bool
}

func clusterByTime(times []time.Time, threshold time.Duration) map[int]clusterStep {
	if len(times) == 0 {
		return nil
	}

	// first pass: assign group ids
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

	// second pass: count group sizes
	groupSizes := map[int]int{}
	for _, gid := range groupIDs {
		groupSizes[gid]++
	}

	// third pass: assign positions within each group
		positions := map[int]int{}
	result := map[int]clusterStep{}

	for i, gid := range groupIDs {
		pos := positions[gid]
		positions[gid] = pos + 1

		if groupSizes[gid] >= 3 {
			result[i] = clusterStep{
				Size:      groupSizes[gid],
				Position:  pos,
				Highlight: true,
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
	return t.Format("Jan 02 15:04")
}
