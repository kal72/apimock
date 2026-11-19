package mcp

import (
	"apimock/internal/config"
	"apimock/internal/domain"
	"apimock/internal/usecase"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

type memoryRepo struct {
	workspaces []domain.Workspace
	endpoints  map[string][]domain.Endpoint
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{
		endpoints: make(map[string][]domain.Endpoint),
	}
}

func (m *memoryRepo) Workspaces(ctx context.Context) ([]domain.Workspace, error) {
	return m.workspaces, nil
}

func (m *memoryRepo) SaveWorkspace(ctx context.Context, ws domain.Workspace) error {
	for i, w := range m.workspaces {
		if w.ID == ws.ID {
			m.workspaces[i] = ws
			return nil
		}
	}
	m.workspaces = append(m.workspaces, ws)
	return nil
}

func (m *memoryRepo) DeleteWorkspace(ctx context.Context, id string) error {
	filtered := make([]domain.Workspace, 0, len(m.workspaces))
	for _, w := range m.workspaces {
		if w.ID != id {
			filtered = append(filtered, w)
		}
	}
	m.workspaces = filtered
	delete(m.endpoints, id)
	return nil
}

func (m *memoryRepo) Endpoints(ctx context.Context, wid string) ([]domain.Endpoint, error) {
	return m.endpoints[wid], nil
}

func (m *memoryRepo) SaveEndpoint(ctx context.Context, ep domain.Endpoint) error {
	eps := m.endpoints[ep.WorkspaceID]
	for i, e := range eps {
		if e.ID == ep.ID {
			eps[i] = ep
			m.endpoints[ep.WorkspaceID] = eps
			return nil
		}
	}
	m.endpoints[ep.WorkspaceID] = append(eps, ep)
	return nil
}

func (m *memoryRepo) DeleteEndpoint(ctx context.Context, wid, eid string) error {
	eps := m.endpoints[wid]
	filtered := make([]domain.Endpoint, 0, len(eps))
	for _, e := range eps {
		if e.ID != eid {
			filtered = append(filtered, e)
		}
	}
	m.endpoints[wid] = filtered
	return nil
}

func setupTestServer() (*Server, *memoryRepo) {
	repo := newMemoryRepo()
	repo.workspaces = []domain.Workspace{
		{ID: "ws-1", Name: "Default Store", Slug: "default-store", PublicDocs: true},
	}
	service := usecase.NewService(repo)
	cfg := config.Config{
		PublicBaseURL: "http://localhost:8080",
		Environment:   "development",
	}
	server := NewServer(service, cfg)
	return server, repo
}

func TestMCPInitializeAndPing(t *testing.T) {
	server, _ := setupTestServer()
	ctx := context.Background()

	// 1. Initialize
	initReq := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  json.RawMessage(`{"protocolVersion":"2024-11-05","clientInfo":{"name":"test-client","version":"1.0"}}`),
	}
	resp := server.HandleRequest(ctx, initReq)
	if resp == nil || resp.Error != nil {
		t.Fatalf("expected successful initialize response, got: %+v", resp)
	}
	initResult, ok := resp.Result.(InitializeResult)
	if !ok {
		t.Fatalf("unexpected result type: %T", resp.Result)
	}
	if initResult.ServerInfo.Name != "mocklab" {
		t.Errorf("expected server name 'mocklab', got %q", initResult.ServerInfo.Name)
	}

	// 2. Notification initialized
	notifReq := Request{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}
	if notifResp := server.HandleRequest(ctx, notifReq); notifResp != nil {
		t.Fatalf("expected nil response for notification, got: %+v", notifResp)
	}

	// 3. Ping
	pingReq := Request{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "ping",
	}
	pingResp := server.HandleRequest(ctx, pingReq)
	if pingResp == nil || pingResp.Error != nil {
		t.Fatalf("expected successful ping, got: %+v", pingResp)
	}
}

func TestMCPToolsList(t *testing.T) {
	server, _ := setupTestServer()
	ctx := context.Background()

	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}
	resp := server.HandleRequest(ctx, req)
	if resp == nil || resp.Error != nil {
		t.Fatalf("expected successful tools/list, got: %+v", resp)
	}
	m, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any result, got %T", resp.Result)
	}
	tools, ok := m["tools"].([]Tool)
	if !ok {
		t.Fatalf("expected []Tool, got %T", m["tools"])
	}
	if len(tools) != 9 {
		t.Fatalf("expected 9 tools, got %d", len(tools))
	}

	expectedTools := map[string]bool{
		"list_workspaces":         false,
		"list_endpoints":          false,
		"get_endpoint":            false,
		"create_endpoint":         false,
		"add_scenario":            false,
		"update_default_response": false,
		"delete_endpoint":         false,
		"test_mock":               false,
		"get_workspace_spec":      false,
	}
	for _, tool := range tools {
		expectedTools[tool.Name] = true
	}
	for name, found := range expectedTools {
		if !found {
			t.Errorf("expected tool %q not found", name)
		}
	}
}

func TestMCPEndpointLifecycleAndScenario(t *testing.T) {
	server, _ := setupTestServer()
	ctx := context.Background()

	// 1. List workspaces
	listWsReq := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"list_workspaces","arguments":{}}`),
	}
	listWsResp := server.HandleRequest(ctx, listWsReq)
	if listWsResp == nil || listWsResp.Error != nil {
		t.Fatalf("list_workspaces failed: %+v", listWsResp)
	}
	res1 := listWsResp.Result.(CallToolResult)
	if res1.IsError || len(res1.Content) == 0 {
		t.Fatalf("unexpected call tool error: %+v", res1)
	}
	if !strings.Contains(res1.Content[0].Text, "default-store") {
		t.Errorf("workspace slug not in response: %s", res1.Content[0].Text)
	}

	// 2. Create endpoint
	createEpReq := Request{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/call",
		Params: json.RawMessage(`{
			"name": "create_endpoint",
			"arguments": {
				"workspace": "default-store",
				"method": "GET",
				"path": "/products",
				"name": "Get Products List",
				"status": 200,
				"body": "{\"items\":[]}"
			}
		}`),
	}
	createEpResp := server.HandleRequest(ctx, createEpReq)
	if createEpResp == nil || createEpResp.Error != nil {
		t.Fatalf("create_endpoint failed: %+v", createEpResp)
	}
	res2 := createEpResp.Result.(CallToolResult)
	if res2.IsError {
		t.Fatalf("create_endpoint returned error: %s", res2.Content[0].Text)
	}

	var createdData struct {
		Endpoint struct {
			ID   string `json:"id"`
			Path string `json:"path"`
		} `json:"endpoint"`
	}
	if err := json.Unmarshal([]byte(res2.Content[0].Text), &createdData); err != nil {
		t.Fatalf("failed to parse create endpoint response: %v", err)
	}
	epID := createdData.Endpoint.ID
	if epID == "" {
		t.Fatalf("expected non-empty endpoint ID")
	}

	// 3. Add scenario
	addScReq := Request{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params: json.RawMessage(fmt.Sprintf(`{
			"name": "add_scenario",
			"arguments": {
				"workspace": "default-store",
				"endpoint_id": "%s",
				"name": "Out of Stock Error",
				"status": 404,
				"body": "{\"error\":\"product not found\"}",
				"condition_type": "query",
				"field": "id",
				"operator": "equals",
				"value": "999"
			}
		}`, epID)),
	}
	addScResp := server.HandleRequest(ctx, addScReq)
	if addScResp == nil || addScResp.Error != nil {
		t.Fatalf("add_scenario failed: %+v", addScResp)
	}
	res3 := addScResp.Result.(CallToolResult)
	if res3.IsError {
		t.Fatalf("add_scenario returned error: %s", res3.Content[0].Text)
	}

	// 4. Test mock: evaluate matching scenario
	testMockReq1 := Request{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "tools/call",
		Params: json.RawMessage(fmt.Sprintf(`{
			"name": "test_mock",
			"arguments": {
				"workspace": "default-store",
				"endpoint_id": "%s",
				"query": {"id": ["999"]}
			}
		}`, epID)),
	}
	testMockResp1 := server.HandleRequest(ctx, testMockReq1)
	if testMockResp1 == nil || testMockResp1.Error != nil {
		t.Fatalf("test_mock failed: %+v", testMockResp1)
	}
	res4 := testMockResp1.Result.(CallToolResult)
	if res4.IsError {
		t.Fatalf("test_mock returned error: %s", res4.Content[0].Text)
	}
	if !strings.Contains(res4.Content[0].Text, `"status": 404`) {
		t.Errorf("expected status 404 in test_mock, got: %s", res4.Content[0].Text)
	}

	// 5. Test mock: evaluate fallback scenario
	testMockReq2 := Request{
		JSONRPC: "2.0",
		ID:      5,
		Method:  "tools/call",
		Params: json.RawMessage(fmt.Sprintf(`{
			"name": "test_mock",
			"arguments": {
				"workspace": "default-store",
				"endpoint_id": "%s",
				"query": {"id": ["123"]}
			}
		}`, epID)),
	}
	testMockResp2 := server.HandleRequest(ctx, testMockReq2)
	res5 := testMockResp2.Result.(CallToolResult)
	if !strings.Contains(res5.Content[0].Text, `"status": 200`) {
		t.Errorf("expected fallback status 200 in test_mock, got: %s", res5.Content[0].Text)
	}

	// 6. Get Workspace Spec
	specReq := Request{
		JSONRPC: "2.0",
		ID:      6,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"get_workspace_spec","arguments":{"workspace":"default-store"}}`),
	}
	specResp := server.HandleRequest(ctx, specReq)
	res6 := specResp.Result.(CallToolResult)
	if !strings.Contains(res6.Content[0].Text, "/products") {
		t.Errorf("expected /products in spec, got: %s", res6.Content[0].Text)
	}

	// 7. Delete endpoint
	delReq := Request{
		JSONRPC: "2.0",
		ID:      7,
		Method:  "tools/call",
		Params: json.RawMessage(fmt.Sprintf(`{
			"name": "delete_endpoint",
			"arguments": {
				"workspace": "default-store",
				"endpoint_id": "%s"
			}
		}`, epID)),
	}
	delResp := server.HandleRequest(ctx, delReq)
	res7 := delResp.Result.(CallToolResult)
	if res7.IsError {
		t.Fatalf("delete_endpoint returned error: %s", res7.Content[0].Text)
	}
}

func TestMCPSessionAndBatch(t *testing.T) {
	server, _ := setupTestServer()
	ctx := context.Background()

	// Session management
	sess := server.CreateSession()
	if sess.ID == "" {
		t.Fatal("expected session ID to be non-empty")
	}
	gotSess, ok := server.GetSession(sess.ID)
	if !ok || gotSess.ID != sess.ID {
		t.Fatalf("failed to retrieve created session")
	}

	sent := sess.Send([]byte(`{"test":"message"}`))
	if !sent {
		t.Fatal("expected message to be sent")
	}

	server.CloseSession(sess.ID)
	_, stillExists := server.GetSession(sess.ID)
	if stillExists {
		t.Fatal("session should have been removed")
	}

	// Batch message
	batchRaw := []byte(`[
		{"jsonrpc":"2.0","id":1,"method":"ping"},
		{"jsonrpc":"2.0","id":2,"method":"ping"}
	]`)
	responses, err := server.HandleMessage(ctx, batchRaw)
	if err != nil {
		t.Fatalf("unexpected error on batch: %v", err)
	}
	if len(responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(responses))
	}
}
