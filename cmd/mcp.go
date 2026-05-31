package cmd

import (
	"github.com/rsteube/gh-slop/pkg/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:    "mcp",
	Short:  "Start the MCP server",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		server := mcp.NewServer(mcp.ToolHandler)
		return server.ServeStdio()
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
