package mcp

import (
	"apimock/internal/config"
	"apimock/internal/usecase"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

type ToolHandler func(ctx context.Context, args json.RawMessage) (CallToolResult, error)

type Session struct {
	ID        string
	Messages  chan []byte
	CreatedAt time.Time
	closed    bool
	mu        sync.Mutex
}

func (s *Session) Send(data []byte) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	select {
	case s.Messages <- data:
		return true
	default:
		slog.Warn("mcp session channel full, dropping message", "session", s.ID)
		return false
	}
}

func (s *Session) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.closed = true
		close(s.Messages)
	}
}

type Server struct {
	service  usecase.Service
	config   config.Config
	tools    map[string]Tool
	handlers map[string]ToolHandler
	toolsMu  sync.RWMutex

	sessions map[string]*Session
	sessMu   sync.RWMutex
}

func NewServer(service usecase.Service, cfg config.Config) *Server {
	s := &Server{
		service:  service,
		config:   cfg,
		tools:    make(map[string]Tool),
		handlers: make(map[string]ToolHandler),
		sessions: make(map[string]*Session),
	}
	s.registerDefaultTools()
	return s
}

func (s *Server) baseURL() string {
	if s.config.PublicBaseURL != "" {
		return strings.TrimRight(s.config.PublicBaseURL, "/")
	}
	return "http://localhost:8080"
}

func (s *Server) RegisterTool(tool Tool, handler ToolHandler) {
	s.toolsMu.Lock()
	defer s.toolsMu.Unlock()
	s.tools[tool.Name] = tool
	s.handlers[tool.Name] = handler
}

func (s *Server) Tools() []Tool {
	s.toolsMu.RLock()
	defer s.toolsMu.RUnlock()
	list := make([]Tool, 0, len(s.tools))
	for _, t := range s.tools {
		list = append(list, t)
	}
	return list
}

func (s *Server) CreateSession() *Session {
	s.sessMu.Lock()
	defer s.sessMu.Unlock()

	b := make([]byte, 16)
	_, _ = rand.Read(b)
	id := hex.EncodeToString(b)

	sess := &Session{
		ID:        id,
		Messages:  make(chan []byte, 64),
		CreatedAt: time.Now(),
	}
	s.sessions[id] = sess
	return sess
}

func (s *Server) GetSession(id string) (*Session, bool) {
	s.sessMu.RLock()
	defer s.sessMu.RUnlock()
	sess, ok := s.sessions[id]
	return sess, ok
}

func (s *Server) CloseSession(id string) {
	s.sessMu.Lock()
	sess, ok := s.sessions[id]
	if ok {
		delete(s.sessions, id)
	}
	s.sessMu.Unlock()

	if ok && sess != nil {
		sess.Close()
	}
}

func (s *Server) HandleRequest(ctx context.Context, req Request) *Response {
	switch req.Method {
	case "initialize":
		return &Response{
			JSONRPC: JSONRPCVersion,
			ID:      req.ID,
			Result: InitializeResult{
				ProtocolVersion: LatestProtocolVer,
				Capabilities: ServerCapabilities{
					Tools:     &ToolsCapability{ListChanged: false},
					Resources: &ResourcesCapability{Subscribe: false, ListChanged: false},
				},
				ServerInfo: ServerInfo{
					Name:    "mocklab",
					Version: "1.0.0",
				},
			},
		}

	case "notifications/initialized":
		return nil

	case "ping":
		return &Response{
			JSONRPC: JSONRPCVersion,
			ID:      req.ID,
			Result:  map[string]any{},
		}

	case "tools/list":
		return &Response{
			JSONRPC: JSONRPCVersion,
			ID:      req.ID,
			Result: map[string]any{
				"tools": s.Tools(),
			},
		}

	case "tools/call":
		var params CallToolParams
		if len(req.Params) > 0 {
			if err := json.Unmarshal(req.Params, &params); err != nil {
				return &Response{
					JSONRPC: JSONRPCVersion,
					ID:      req.ID,
					Error: &RPCError{
						Code:    CodeInvalidParams,
						Message: "Invalid params: " + err.Error(),
					},
				}
			}
		}

		s.toolsMu.RLock()
		handler, ok := s.handlers[params.Name]
		s.toolsMu.RUnlock()

		if !ok {
			return &Response{
				JSONRPC: JSONRPCVersion,
				ID:      req.ID,
				Result: CallToolResult{
					Content: []ContentItem{
						{Type: "text", Text: fmt.Sprintf("Tool '%s' not found", params.Name)},
					},
					IsError: true,
				},
			}
		}

		res, err := handler(ctx, params.Arguments)
		if err != nil {
			res = CallToolResult{
				Content: []ContentItem{
					{Type: "text", Text: "Error executing tool: " + err.Error()},
				},
				IsError: true,
			}
		}

		return &Response{
			JSONRPC: JSONRPCVersion,
			ID:      req.ID,
			Result:  res,
		}

	case "resources/list":
		workspaces, err := s.service.Workspaces(ctx)
		if err != nil {
			return &Response{
				JSONRPC: JSONRPCVersion,
				ID:      req.ID,
				Error:   &RPCError{Code: CodeInternalError, Message: err.Error()},
			}
		}
		resources := make([]Resource, 0, len(workspaces))
		for _, w := range workspaces {
			resources = append(resources, Resource{
				URI:         fmt.Sprintf("mocklab://workspace/%s/spec", w.Slug),
				Name:        fmt.Sprintf("%s Specification", w.Name),
				Description: fmt.Sprintf("Full API mock specification for workspace '%s'", w.Name),
				MIMEType:    "application/json",
			})
		}
		return &Response{
			JSONRPC: JSONRPCVersion,
			ID:      req.ID,
			Result: map[string]any{
				"resources": resources,
			},
		}

	case "resources/read":
		var params ReadResourceParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &Response{
				JSONRPC: JSONRPCVersion,
				ID:      req.ID,
				Error:   &RPCError{Code: CodeInvalidParams, Message: err.Error()},
			}
		}
		if strings.HasPrefix(params.URI, "mocklab://workspace/") && strings.HasSuffix(params.URI, "/spec") {
			slug := strings.TrimPrefix(params.URI, "mocklab://workspace/")
			slug = strings.TrimSuffix(slug, "/spec")
			ws, err := s.resolveWorkspace(ctx, slug)
			if err != nil {
				return &Response{
					JSONRPC: JSONRPCVersion,
					ID:      req.ID,
					Error:   &RPCError{Code: CodeInvalidParams, Message: err.Error()},
				}
			}
			endpoints, err := s.service.Endpoints(ctx, ws.ID)
			if err != nil {
				return &Response{
					JSONRPC: JSONRPCVersion,
					ID:      req.ID,
					Error:   &RPCError{Code: CodeInternalError, Message: err.Error()},
				}
			}
			base := s.baseURL()
			out := map[string]any{
				"workspace": ws,
				"endpoints": endpoints,
				"base_url":  fmt.Sprintf("%s/public/%s", base, ws.Slug),
			}
			b, _ := json.MarshalIndent(out, "", "  ")
			return &Response{
				JSONRPC: JSONRPCVersion,
				ID:      req.ID,
				Result: ReadResourceResult{
					Contents: []ResourceContent{
						{
							URI:      params.URI,
							MIMEType: "application/json",
							Text:     string(b),
						},
					},
				},
			}
		}
		return &Response{
			JSONRPC: JSONRPCVersion,
			ID:      req.ID,
			Error:   &RPCError{Code: CodeInvalidParams, Message: fmt.Sprintf("Unsupported resource URI: %s", params.URI)},
		}

	default:
		if req.ID == nil {
			return nil
		}
		return &Response{
			JSONRPC: JSONRPCVersion,
			ID:      req.ID,
			Error: &RPCError{
				Code:    CodeMethodNotFound,
				Message: fmt.Sprintf("Method '%s' not found", req.Method),
			},
		}
	}
}

func (s *Server) HandleMessage(ctx context.Context, raw []byte) ([]*Response, error) {
	var req Request
	if err := json.Unmarshal(raw, &req); err == nil && req.Method != "" {
		resp := s.HandleRequest(ctx, req)
		if resp == nil {
			return nil, nil
		}
		return []*Response{resp}, nil
	}

	var batch []Request
	if err := json.Unmarshal(raw, &batch); err == nil && len(batch) > 0 {
		var responses []*Response
		for _, r := range batch {
			if resp := s.HandleRequest(ctx, r); resp != nil {
				responses = append(responses, resp)
			}
		}
		return responses, nil
	}

	return []*Response{
		{
			JSONRPC: JSONRPCVersion,
			Error: &RPCError{
				Code:    CodeParseError,
				Message: "Parse error: expected valid JSON-RPC 2.0 object or batch",
			},
		},
	}, nil
}
