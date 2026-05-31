package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/rsteube/gh-slop/pkg/slop"
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
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ServerInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
}

type ToolCallHandler func(args json.RawMessage) (any, *Error)

func NewServer(toolHandler ToolCallHandler) *Server {
	return &Server{
		tools: []Tool{
			{
				Name:        "list-repos",
				Description: "List repositories (current or provided)",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {}
					}
				}`),
			},
			{
				Name:        "list-sloppers",
				Description: "List open pull requests from new or low-contribution authors (sloppers)",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"repositories": {
							"description": "List of repositories to check (owner/repo format). If not provided, uses the current repository.",
							"type": "array",
							"items": {
								"type": "string"
							}
						},
						"min_contributions": {
							"description": "Minimum number of merged PRs to not be considered a new contributor (default: 1)",
							"type": "integer",
							"default": 1
						}
					}
				}`),
			},
		},
		toolHandler: toolHandler,
	}
}

func (s *Server) ServeStdio() error {
	return s.ServeConn(os.Stdin, os.Stdout)
}

func (s *Server) ServeConn(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	var mu sync.Mutex

	for scanner.Scan() {
		var req Request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}

		var resp Response
		resp.JSONRPC = "2.0"
		resp.ID = req.ID

		switch req.Method {
		case "initialize":
			resp.Result = InitializeResult{
				ProtocolVersion: "2024-11-05",
				Capabilities:    map[string]interface{}{"tools": map[string]interface{}{}},
				ServerInfo: struct {
					Name    string `json:"name"`
					Version string `json:"version"`
				}{
					Name:    "gh-slop",
					Version: "0.0.0",
				},
			}
		case "notifications/initialized":
			continue
		case "tools/list":
			resp.Result = map[string]interface{}{"tools": s.tools}
		case "tools/call":
			resp.Result, resp.Error = s.toolHandler(req.Params)
		default:
			resp.Error = &Error{Code: -32601, Message: "Method not found"}
		}

		mu.Lock()
		data, _ := json.Marshal(resp)
		fmt.Fprintln(w, string(data))
		mu.Unlock()
	}

	return scanner.Err()
}

func ToolHandler(params json.RawMessage) (any, *Error) {
	var args struct {
		Name string `json:"name"`
	}

	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &Error{Code: -32602, Message: "Invalid params"}
	}

	switch args.Name {
	case "list-repos":
		return ListReposHandler(params)
	case "list-sloppers":
		return ListSloppersHandler(params)
	default:
		return nil, &Error{Code: -32602, Message: "Unknown tool: " + args.Name}
	}
}

func ListReposHandler(params json.RawMessage) (any, *Error) {
	var args struct {
		Name      string   `json:"name"`
		Arguments struct{} `json:"arguments"`
	}

	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &Error{Code: -32602, Message: "Invalid params"}
	}

	if args.Name != "list-repos" {
		return nil, &Error{Code: -32602, Message: "Unknown tool: " + args.Name}
	}

	repos, err := slop.Repos(nil)
	if err != nil {
		return nil, &Error{Code: -32603, Message: err.Error()}
	}

	var out string
	for _, r := range repos {
		out += fmt.Sprintf("%s/%s\n", r.Owner, r.Name)
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": out},
		},
	}, nil
}

func ListSloppersHandler(params json.RawMessage) (any, *Error) {
	var args struct {
		Name      string `json:"name"`
		Arguments struct {
			Repositories     []string `json:"repositories"`
			MinContributions int      `json:"min_contributions"`
		} `json:"arguments"`
	}

	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &Error{Code: -32602, Message: "Invalid params"}
	}

	if args.Name != "list-sloppers" {
		return nil, &Error{Code: -32602, Message: "Unknown tool: " + args.Name}
	}

	repos, err := slop.Repos(args.Arguments.Repositories)
	if err != nil {
		return nil, &Error{Code: -32603, Message: err.Error()}
	}
	if len(repos) == 0 {
		return nil, &Error{Code: -32603, Message: "no repository found. Please specify repositories as argument"}
	}

	minContrib := args.Arguments.MinContributions
	if minContrib == 0 {
		minContrib = 1
	}

	prs, err := slop.ListNewContributors(repos, minContrib)
	if err != nil {
		return nil, &Error{Code: -32603, Message: err.Error()}
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": formatPRs(prs)},
		},
	}, nil
}

func formatPRs(prs []slop.PRWithRepo) string {
	if len(prs) == 0 {
		return "No new contributors found."
	}
	var out string
	for _, pr := range prs {
		out += fmt.Sprintf("#%d: %s (@%s)\n", pr.PullRequest.Number, pr.PullRequest.Title, pr.PullRequest.Author)
	}
	return out
}
