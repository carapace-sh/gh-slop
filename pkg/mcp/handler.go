package mcp

import (
	"encoding/json"
	"errors"
	"io"
	"os"
)

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
	resp := Response{
		JSONRPC: "2.0",
		ID:      request.ID,
	}

	if request.Method == "notifications/initialized" {
		return Response{}, false
	}

	if len(request.ID) == 0 {
		return Response{}, false
	}

	switch request.Method {
	case "initialize":
		resp.Result = InitializeResult{
			ProtocolVersion: "2025-11-25",
			Capabilities:    ServerCaps{Tools: map[string]any{}},
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
		resp.Result, resp.Error = s.handleToolCall(request.Params)
	default:
		resp.Error = &Error{Code: -32601, Message: "Method not found"}
	}

	return resp, true
}

func (s *Server) handleToolCall(params json.RawMessage) (any, *Error) {
	var args struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &Error{Code: -32602, Message: "Invalid params"}
	}

	var content string
	var isErr bool

	switch args.Name {
	case "list-repos":
		content, isErr = listReposHandler(args.Arguments)
	case "list-sloppers":
		content, isErr = listSloppersHandler(args.Arguments)
	case "profile-sloppers":
		content, isErr = profileSloppersHandler(args.Arguments)
	case "view-prs":
		content, isErr = viewPRsHandler(args.Arguments)
	case "close-prs":
		content, isErr = closePRsHandler(args.Arguments)
	case "view-issues":
		content, isErr = viewIssuesHandler(args.Arguments)
	case "list-issues":
		content, isErr = listIssuesHandler(args.Arguments)
	default:
		return nil, &Error{Code: -32602, Message: "Unknown tool: " + args.Name}
	}

	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": content},
		},
		"isError": isErr,
	}, nil
}
