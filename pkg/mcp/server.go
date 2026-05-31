package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

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
		},
		toolHandler: toolHandler,
	}
}

func (s *Server) ServeStdio() error {
	return s.ServeConn(os.Stdin, os.Stdout)
}

func (s *Server) ServeConn(r io.Reader, w io.Writer) error {
	decoder := json.NewDecoder(r)
	encoder := json.NewEncoder(w)

	for {
		var message json.RawMessage
		if err := decoder.Decode(&message); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		result, err := s.processMessage(message)
		if err != nil {
			return err
		}
		if result != nil {
			if err := encoder.Encode(result); err != nil {
				return err
			}
		}
	}
}

func (s *Server) processMessage(message json.RawMessage) (any, error) {
	switch firstNonSpace(message) {
	case '[':
		var requests []Request
		if err := json.Unmarshal(message, &requests); err != nil {
			return nil, err
		}
		responses := make([]Response, 0, len(requests))
		for _, request := range requests {
			response, ok := s.handleRequest(request)
			if ok {
				responses = append(responses, response)
			}
		}
		if len(responses) == 0 {
			return nil, nil
		}
		return responses, nil
	default:
		var request Request
		if err := json.Unmarshal(message, &request); err != nil {
			return nil, err
		}
		response, ok := s.handleRequest(request)
		if !ok {
			return nil, nil
		}
		return response, nil
	}
}

func firstNonSpace(message []byte) byte {
	for _, b := range message {
		switch b {
		case ' ', '\n', '\r', '\t':
			continue
		default:
			return b
		}
	}
	return 0
}

func (s *Server) handleRequest(request Request) (Response, bool) {
	if len(request.ID) == 0 {
		return Response{}, false
	}

	var resp Response
	resp.JSONRPC = "2.0"
	resp.ID = request.ID

	switch request.Method {
	case "initialize":
		resp.Result = InitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities:    map[string]any{"tools": map[string]any{}},
			ServerInfo: struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			}{
				Name:    "gh-slop",
				Version: "0.0.0",
			},
		}
	case "tools/list":
		resp.Result = map[string]any{"tools": s.tools}
	case "tools/call":
		resp.Result, resp.Error = s.toolHandler(request.Params)
	default:
		resp.Error = &Error{Code: -32601, Message: "Method not found"}
	}

	return resp, true
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
	case "profile-sloppers":
		return ProfileSloppersHandler(params)
	case "view-prs":
		return SlopPRsHandler(params)
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

	repos, err := slop.AccessibleRepos()
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

	repos, err := slop.ResolveRepos(args.Arguments.Repositories)
	if err != nil {
		return nil, &Error{Code: -32603, Message: err.Error()}
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

func ProfileSloppersHandler(params json.RawMessage) (any, *Error) {
	var args struct {
		Name      string `json:"name"`
		Arguments struct {
			Sloppers []string `json:"sloppers"`
		} `json:"arguments"`
	}

	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &Error{Code: -32602, Message: "Invalid params"}
	}

	if args.Name != "profile-sloppers" {
		return nil, &Error{Code: -32602, Message: "Unknown tool: " + args.Name}
	}

	if len(args.Arguments.Sloppers) == 0 {
		return nil, &Error{Code: -32602, Message: "sloppers is required"}
	}

	profiles, err := slop.FetchUserProfiles(args.Arguments.Sloppers)
	if err != nil {
		return nil, &Error{Code: -32603, Message: err.Error()}
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": formatProfiles(profiles)},
		},
	}, nil
}

func formatProfiles(profiles []slop.UserProfile) string {
	var b strings.Builder
	for i, p := range profiles {
		if i > 0 {
			b.WriteString("\n")
		}
		mergeRate := 0
		if p.TotalPRs > 0 {
			mergeRate = p.MergedPRs * 100 / p.TotalPRs
		}
		fmt.Fprintf(&b, "## @%s\n", p.Login)
		fmt.Fprintf(&b, "Account created: %s\n", p.CreatedAt.Format("2006-01-02"))
		fmt.Fprintf(&b, "Total commits: %d\n", p.TotalCommits)
		fmt.Fprintf(&b, "Total PRs: %d (merged: %d, open: %d, closed: %d) — %d%% merge rate\n", p.TotalPRs, p.MergedPRs, p.OpenPRs, p.ClosedPRs, mergeRate)
		fmt.Fprintf(&b, "Repos targeted: %d\n", p.TotalReposTargeted)
		if len(p.PRs) > 0 {
			b.WriteString("Recent PRs:\n")
			for _, pr := range p.PRs {
				fmt.Fprintf(&b, "  - [%s] %s (%s)\n", pr.State, pr.Title, pr.Repo)
			}
		}
	}
	return b.String()
}

func SlopPRsHandler(params json.RawMessage) (any, *Error) {
	var args struct {
		Name      string `json:"name"`
		Arguments struct {
			PRs []string `json:"prs"`
		} `json:"arguments"`
	}

	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &Error{Code: -32602, Message: "Invalid params"}
	}

	if args.Name != "view-prs" {
		return nil, &Error{Code: -32602, Message: "Unknown tool: " + args.Name}
	}

	if len(args.Arguments.PRs) == 0 {
		return nil, &Error{Code: -32602, Message: "prs is required"}
	}

	details, err := slop.FetchPRDetails(args.Arguments.PRs)
	if err != nil {
		return nil, &Error{Code: -32603, Message: err.Error()}
	}

	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": formatPRDetails(details)},
		},
	}, nil
}

func formatPRDetails(details []slop.PRDetail) string {
	var b strings.Builder
	for i, d := range details {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "## %s#%d\n", d.Repo, d.Number)
		fmt.Fprintf(&b, "Title: %s\n", d.Title)
		fmt.Fprintf(&b, "Author: @%s\n", d.Author)
		fmt.Fprintf(&b, "Created: %s\n", d.CreatedAt)
		fmt.Fprintf(&b, "URL: %s\n", d.URL)
		if d.Body != "" {
			b.WriteString("---\n")
			b.WriteString(d.Body)
			b.WriteString("\n")
		}
	}
	return b.String()
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
