package cmd

import (
	"github.com/carapace-sh/carapace"
	"github.com/rsteube/gh-slop/pkg/crush"
	"github.com/spf13/cobra"
)

var detectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Detect slop contributions using Crush AI analysis",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return crush.RunDetect(cmd.Context(), repos)
	},
}

func init() {
	rootCmd.AddCommand(detectCmd)
	carapace.Gen(detectCmd)
}
