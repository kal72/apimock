package handler

import (
	"apimock/internal/config"
	"apimock/internal/domain"
	"apimock/internal/mcp"
	"apimock/internal/usecase"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type mcpTestRepo struct {
	workspaces []domain.Workspace
	endpoints  map[string][]domain.Endpoint
}

func (m *mcpTestRepo) Workspaces(ctx context.Context) ([]domain.Workspace, error) {
	return m.workspaces, nil
}
func (m *mcpTestRepo) SaveWorkspace(ctx context.Context, ws domain.Workspace) error {
	m.workspaces = append(m.workspaces, ws)
	return nil
}
func (m *mcpTestRepo) DeleteWorkspace(ctx context.Context, id string) error {
	return nil
}
func (m *mcpTestRepo) Endpoints(ctx context.Context, wid string) ([]domain.Endpoint, error) {
	return m.endpoints[wid], nil
}
func (m *mcpTestRepo) SaveEndpoint(ctx context.Context, ep domain.Endpoint) error {
	m.endpoints[ep.WorkspaceID] = append(m.endpoints[ep.WorkspaceID], ep)
	return nil
}
func (m *mcpTestRepo) DeleteEndpoint(ctx context.Context, wid, eid string) error {
	return nil
}

type mcpAuthRepo struct {
	apiKeys map[string]domain.User
}

func (m *mcpAuthRepo) HasUsers(context.Context) (bool, error) { return true, nil }
func (m *mcpAuthRepo) CreateFirstUser(context.Context, domain.User) error { return nil }
func (m *mcpAuthRepo) Users(context.Context) ([]domain.User, error) { return nil, nil }
func (m *mcpAuthRepo) UserByUsername(context.Context, string) (domain.User, error) { return domain.User{}, nil }
func (m *mcpAuthRepo) SaveUser(context.Context, domain.User, string, bool) error { return nil }
func (m *mcpAuthRepo) DeleteUser(context.Context, string, string) error { return nil }
func (m *mcpAuthRepo) CreateSession(context.Context, string, string, string, time.Time) error { return nil }
func (m *mcpAuthRepo) SessionUser(context.Context, string) (domain.User, error) { return domain.User{}, nil }
func (m *mcpAuthRepo) DeleteSession(context.Context, string) error { return nil }
func (m *mcpAuthRepo) ChangePassword(context.Context, string, string, string) error { return nil }
func (m *mcpAuthRepo) APIKeys(context.Context, string) ([]domain.APIKey, error) { return nil, nil }
func (m *mcpAuthRepo) CreateAPIKey(context.Context, domain.APIKey) error { return nil }
func (m *mcpAuthRepo) DeleteAPIKey(context.Context, string, string) error { return nil }
func (m *mcpAuthRepo) AuthenticateAPIKey(ctx context.Context, hash string) (domain.User, error) {
	if u, ok := m.apiKeys[hash]; ok {
		return u, nil
	}
	return domain.User{}, usecase.ErrUnauthorized
}

func setupMCPTestHandler(apiKey string, env string) (Handler, *mcpTestRepo) {
	repo := &mcpTestRepo{
		workspaces: []domain.Workspace{
			{ID: "ws-test", Name: "Test Workspace", Slug: "test-workspace"},
		},
		endpoints: make(map[string][]domain.Endpoint),
	}
	authRepo := &mcpAuthRepo{
		apiKeys: make(map[string]domain.User),
	}
	if apiKey != "" {
		authRepo.apiKeys[usecase.TokenHash(apiKey)] = domain.User{
			ID: "u1", Username: "admin", Role: "admin", Active: true,
		}
	}
	cfg := config.Config{
		Environment:   env,
		PublicBaseURL: "http://localhost:8080",
	}
	service := usecase.NewService(repo)
	h := Handler{
		Config:  cfg,
		Service: service,
		Auth:    &Identity{Service: usecase.AuthService{Repo: authRepo}},
		Assets:  http.NotFoundHandler(),
	}
	return h, repo
}

type inMemorySSERecorder struct {
	mu     sync.Mutex
	header http.Header
	buf    bytes.Buffer
	status int
}

func newInMemorySSERecorder() *inMemorySSERecorder {
	return &inMemorySSERecorder{
		header: make(http.Header),
		status: http.StatusOK,
	}
}

func (r *inMemorySSERecorder) Header() http.Header {
	return r.header
}

func (r *inMemorySSERecorder) Write(b []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buf.Write(b)
}

func (r *inMemorySSERecorder) WriteHeader(statusCode int) {
	r.status = statusCode
}

func (r *inMemorySSERecorder) Flush() {}

func (r *inMemorySSERecorder) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buf.String()
}

func TestMCPSSEAndMessagesFlow(t *testing.T) {
	h, _ := setupMCPTestHandler("", "development")
	routes := h.Routes()

	rec := newInMemorySSERecorder()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sseReq := httptest.NewRequest("GET", "/api/mcp/sse", nil).WithContext(ctx)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		routes.ServeHTTP(rec, sseReq)
	}()

	// 1. Wait for endpoint event
	var endpointLine string
	for i := 0; i < 50; i++ {
		text := rec.String()
		if idx := strings.Index(text, "data: /api/mcp/messages?sessionId="); idx != -1 {
			end := strings.Index(text[idx:], "\n")
			if end != -1 {
				endpointLine = strings.TrimSpace(text[idx+len("data: ") : idx+end])
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	if endpointLine == "" {
		t.Fatalf("timed out waiting for endpoint event, got: %s", rec.String())
	}

	// 2. Send ping via POST to the endpoint
	pingBody := `{"jsonrpc":"2.0","id":100,"method":"ping"}`
	postReq := httptest.NewRequest("POST", endpointLine, strings.NewReader(pingBody))
	postReq.Header.Set("Content-Type", "application/json")
	postRec := httptest.NewRecorder()

	routes.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d (body: %s)", postRec.Code, postRec.Body.String())
	}

	// 3. Receive JSON-RPC response on the SSE connection
	var messageData string
	for i := 0; i < 50; i++ {
		text := rec.String()
		if idx := strings.Index(text, `{"jsonrpc":"2.0"`); idx != -1 {
			end := strings.Index(text[idx:], "\n")
			if end != -1 {
				messageData = strings.TrimSpace(text[idx : idx+end])
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	if messageData == "" {
		t.Fatalf("timed out waiting for message data, got: %s", rec.String())
	}

	var rpcResp mcp.Response
	if err := json.Unmarshal([]byte(messageData), &rpcResp); err != nil {
		t.Fatalf("failed to unmarshal JSON-RPC response %q: %v", messageData, err)
	}
	if fmt.Sprintf("%v", rpcResp.ID) != "100" {
		t.Errorf("expected ID 100, got %v", rpcResp.ID)
	}

	cancel()
	wg.Wait()
}

func TestMCPAuthenticationEnforcement(t *testing.T) {
	apiKey := "secret-agent-key-12345"
	h, _ := setupMCPTestHandler(apiKey, "production")
	routes := h.Routes()

	// 1. Unauthorized request without key in production
	req1 := httptest.NewRequest("GET", "http://localhost:8080/api/mcp/sse", nil)
	rec1 := httptest.NewRecorder()
	routes.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rec1.Code)
	}

	// 2. Authorized request with Bearer header
	rec2 := newInMemorySSERecorder()
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	req2 := httptest.NewRequest("GET", "http://localhost:8080/api/mcp/sse", nil).WithContext(ctx2)
	req2.Header.Set("Authorization", "Bearer "+apiKey)

	var wg2 sync.WaitGroup
	wg2.Add(1)
	go func() {
		defer wg2.Done()
		routes.ServeHTTP(rec2, req2)
	}()

	for i := 0; i < 50; i++ {
		if strings.Contains(rec2.String(), "event: endpoint") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !strings.Contains(rec2.String(), "event: endpoint") {
		t.Errorf("expected event: endpoint, got %q", rec2.String())
	}
	cancel2()
	wg2.Wait()

	// 3. Authorized request with ?token= query parameter
	rec3 := newInMemorySSERecorder()
	ctx3, cancel3 := context.WithCancel(context.Background())
	defer cancel3()
	req3 := httptest.NewRequest("GET", "http://localhost:8080/api/mcp/sse?token="+apiKey, nil).WithContext(ctx3)

	var wg3 sync.WaitGroup
	wg3.Add(1)
	go func() {
		defer wg3.Done()
		routes.ServeHTTP(rec3, req3)
	}()

	for i := 0; i < 50; i++ {
		if strings.Contains(rec3.String(), "event: endpoint") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !strings.Contains(rec3.String(), "event: endpoint") {
		t.Errorf("expected event: endpoint, got %q", rec3.String())
	}
	cancel3()
	wg3.Wait()
}

func TestMCPCORSPreflight(t *testing.T) {
	h, _ := setupMCPTestHandler("", "development")
	routes := h.Routes()

	req := httptest.NewRequest("OPTIONS", "/api/mcp/sse", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()

	routes.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content for CORS preflight, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Errorf("expected Access-Control-Allow-Origin to be http://localhost:3000, got %q", got)
	}
}
