package mcp

import (
	"encoding/json"
)

type Server struct {
	tools       []Tool
	toolHandler ToolCallHandler
}

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type InitializeResult struct {
	ProtocolVersion string      `json:"protocolVersion"`
	Capabilities    ServerCaps `json:"capabilities"`
	ServerInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
}

type ServerCaps struct {
	Tools map[string]any `json:"tools,omitempty"`
}

type ToolCallHandler func(args json.RawMessage) (any, *Error)

func NewServer(toolHandler ToolCallHandler) *Server {
	return &Server{
		tools: []Tool{
			{
				Name:        "list-repos",
				Description: "List repositories (current or provided)",
				InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
			},
			{
				Name:        "list-sloppers",
				Description: "List open pull requests from new or low-contribution authors (sloppers)",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"repositories":{"description":"List of repositories to check (owner/repo format). If not provided, uses the current repository.","type":"array","items":{"type":"string"}},"min_contributions":{"description":"Minimum number of merged PRs to not be considered a new contributor (default: 1)","type":"integer","default":1}}}`),
			},
			{
				Name:        "profile-sloppers",
				Description: "Fetch detailed GitHub profiles for multiple authors in a single batch call, returning account age, commit count, PR distribution, merge rate, and recent PRs",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"sloppers":{"description":"List of GitHub usernames to profile","type":"array","items":{"type":"string"}}},"required":["sloppers"]}`),
			},
			{
				Name:        "view-prs",
				Description: "Fetch details (title, body, author, createdAt, URL) for a list of PRs in a single optimized batch call, instead of making individual requests per PR",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"prs":{"description":"List of PR references in OWNER/REPO#NUMBER format (e.g. [\"cli/cli#1234\", \"owner/repo#567\"])","type":"array","items":{"type":"string"}}},"required":["prs"]}`),
			},
			{
				Name:        "close-prs",
				Description: "Close pull requests by PR reference, returning the new state of each PR after closing",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"prs":{"description":"List of PR references in OWNER/REPO#NUMBER format (e.g. [\"cli/cli#1234\", \"owner/repo#567\"])","type":"array","items":{"type":"string"}}},"required":["prs"]}`),
			},
		},
		toolHandler: toolHandler,
	}
}
