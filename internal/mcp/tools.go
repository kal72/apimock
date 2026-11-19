package mcp

import (
	"apimock/internal/domain"
	"apimock/internal/usecase"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func (s *Server) registerDefaultTools() {
	s.RegisterTool(Tool{
		Name:        "list_workspaces",
		Description: "List all Mocklab workspaces available. Returns workspace IDs, names, slugs, and base URLs.",
		InputSchema: ToolInputSchema{
			Type:       "object",
			Properties: map[string]PropertySchema{},
		},
	}, s.toolListWorkspaces)

	s.RegisterTool(Tool{
		Name:        "list_endpoints",
		Description: "List all mock REST endpoints in a Mocklab workspace.",
		InputSchema: ToolInputSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"workspace": {
					Type:        "string",
					Description: "Workspace ID or workspace slug (e.g. 'demo-store')",
				},
			},
			Required: []string{"workspace"},
		},
	}, s.toolListEndpoints)

	s.RegisterTool(Tool{
		Name:        "get_endpoint",
		Description: "Get complete details and response scenarios of a specific mock endpoint in Mocklab.",
		InputSchema: ToolInputSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"workspace": {
					Type:        "string",
					Description: "Workspace ID or slug",
				},
				"endpoint_id": {
					Type:        "string",
					Description: "Endpoint ID",
				},
			},
			Required: []string{"workspace", "endpoint_id"},
		},
	}, s.toolGetEndpoint)

	s.RegisterTool(Tool{
		Name:        "create_endpoint",
		Description: "Create a new mock REST API endpoint in Mocklab with an initial HTTP response.",
		InputSchema: ToolInputSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"workspace": {
					Type:        "string",
					Description: "Workspace ID or slug",
				},
				"method": {
					Type:        "string",
					Description: "HTTP method: GET, POST, PUT, PATCH, DELETE, HEAD, or OPTIONS",
					Enum:        []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
				},
				"path": {
					Type:        "string",
					Description: "URL route path starting with '/' (e.g. '/api/v1/users' or '/products/:id')",
				},
				"name": {
					Type:        "string",
					Description: "Human-readable label for the endpoint (e.g. 'Get User by ID')",
				},
				"status": {
					Type:        "integer",
					Description: "Default HTTP response status code (e.g. 200, 201, 204). Default is 200.",
				},
				"body": {
					Type:        "string",
					Description: "Default JSON response payload (e.g. '{\"success\": true}').",
				},
				"delay_ms": {
					Type:        "integer",
					Description: "Simulated network delay in milliseconds (0–10000). Default is 0.",
				},
			},
			Required: []string{"workspace", "method", "path"},
		},
	}, s.toolCreateEndpoint)

	s.RegisterTool(Tool{
		Name:        "add_scenario",
		Description: "Add a priority-based conditional response scenario to an existing endpoint (e.g. return 401 if token missing, 404 if item not found, 402 if card declined).",
		InputSchema: ToolInputSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"workspace": {
					Type:        "string",
					Description: "Workspace ID or slug",
				},
				"endpoint_id": {
					Type:        "string",
					Description: "Endpoint ID",
				},
				"name": {
					Type:        "string",
					Description: "Scenario name (e.g. 'Invalid Token' or 'Payment Declined')",
				},
				"status": {
					Type:        "integer",
					Description: "HTTP status code to return when condition matches (e.g. 400, 401, 403, 404, 500)",
				},
				"body": {
					Type:        "string",
					Description: "JSON response payload to return when condition matches (e.g. '{\"error\": \"Unauthorized\"}')",
				},
				"condition_type": {
					Type:        "string",
					Description: "Which part of the request to inspect",
					Enum:        []string{"header", "query", "path", "body"},
				},
				"field": {
					Type:        "string",
					Description: "Field name to inspect (e.g. 'Authorization', 'role', 'id', 'user.email')",
				},
				"operator": {
					Type:        "string",
					Description: "Matching operator",
					Enum:        []string{"equals", "not_equals", "contains", "exists", "empty"},
				},
				"value": {
					Type:        "string",
					Description: "Expected value for equals, not_equals, or contains checks",
				},
				"delay_ms": {
					Type:        "integer",
					Description: "Simulated response delay in milliseconds (0–10000). Default is 0.",
				},
			},
			Required: []string{"workspace", "endpoint_id", "name", "status", "condition_type", "field", "operator"},
		},
	}, s.toolAddScenario)

	s.RegisterTool(Tool{
		Name:        "update_default_response",
		Description: "Update the default (fallback) response of an endpoint (e.g. to test error handling or loading delay).",
		InputSchema: ToolInputSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"workspace": {
					Type:        "string",
					Description: "Workspace ID or slug",
				},
				"endpoint_id": {
					Type:        "string",
					Description: "Endpoint ID",
				},
				"status": {
					Type:        "integer",
					Description: "New default HTTP status code (200–599)",
				},
				"body": {
					Type:        "string",
					Description: "New default JSON response payload",
				},
				"delay_ms": {
					Type:        "integer",
					Description: "New simulated delay in milliseconds (0–10000)",
				},
			},
			Required: []string{"workspace", "endpoint_id"},
		},
	}, s.toolUpdateDefaultResponse)

	s.RegisterTool(Tool{
		Name:        "delete_endpoint",
		Description: "Delete a mock endpoint from a Mocklab workspace.",
		InputSchema: ToolInputSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"workspace": {
					Type:        "string",
					Description: "Workspace ID or slug",
				},
				"endpoint_id": {
					Type:        "string",
					Description: "Endpoint ID",
				},
			},
			Required: []string{"workspace", "endpoint_id"},
		},
	}, s.toolDeleteEndpoint)

	s.RegisterTool(Tool{
		Name:        "test_mock",
		Description: "Simulate and test a request against a mock endpoint flow. Returns the evaluated status, matched scenario, headers, body, and execution trace.",
		InputSchema: ToolInputSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"workspace": {
					Type:        "string",
					Description: "Workspace ID or slug",
				},
				"endpoint_id": {
					Type:        "string",
					Description: "Endpoint ID",
				},
				"params": {
					Type:        "object",
					Description: "Path parameters map (e.g. {\"id\": \"123\"})",
				},
				"query": {
					Type:        "object",
					Description: "Query parameters map (e.g. {\"page\": [\"1\"], \"search\": [\"laptop\"]})",
				},
				"headers": {
					Type:        "object",
					Description: "Request headers map (e.g. {\"Authorization\": \"Bearer test-token\"})",
				},
				"body": {
					Type:        "object",
					Description: "JSON request body payload object or JSON string",
				},
			},
			Required: []string{"workspace", "endpoint_id"},
		},
	}, s.toolTestMock)

	s.RegisterTool(Tool{
		Name:        "get_workspace_spec",
		Description: "Get a consolidated API contract specification of all mock endpoints in a workspace. Ideal for AI agents to inspect before generating client code.",
		InputSchema: ToolInputSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"workspace": {
					Type:        "string",
					Description: "Workspace ID or slug",
				},
			},
			Required: []string{"workspace"},
		},
	}, s.toolGetWorkspaceSpec)
}

func (s *Server) toolListWorkspaces(ctx context.Context, args json.RawMessage) (CallToolResult, error) {
	workspaces, err := s.service.Workspaces(ctx)
	if err != nil {
		return errorResult("Failed to list workspaces: " + err.Error()), nil
	}
	type wsItem struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		Slug           string `json:"slug"`
		PublicDocs     bool   `json:"public_docs"`
		PublicBaseURL  string `json:"public_base_url"`
		PublicDocsURL  string `json:"public_docs_url"`
	}
	items := make([]wsItem, 0, len(workspaces))
	base := s.baseURL()
	for _, w := range workspaces {
		items = append(items, wsItem{
			ID:            w.ID,
			Name:          w.Name,
			Slug:          w.Slug,
			PublicDocs:    w.PublicDocs,
			PublicBaseURL: fmt.Sprintf("%s/public/%s", base, w.Slug),
			PublicDocsURL: fmt.Sprintf("%s/share/%s", base, w.Slug),
		})
	}
	b, _ := json.MarshalIndent(items, "", "  ")
	return successResult(string(b)), nil
}

func (s *Server) toolListEndpoints(ctx context.Context, args json.RawMessage) (CallToolResult, error) {
	var input struct {
		Workspace string `json:"workspace"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return errorResult("Invalid arguments: " + err.Error()), nil
	}
	ws, err := s.resolveWorkspace(ctx, input.Workspace)
	if err != nil {
		return errorResult(err.Error()), nil
	}
	endpoints, err := s.service.Endpoints(ctx, ws.ID)
	if err != nil {
		return errorResult("Failed to list endpoints: " + err.Error()), nil
	}
	type epItem struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		Method        string `json:"method"`
		Path          string `json:"path"`
		CallableURL   string `json:"callable_url"`
		Scenarios     int    `json:"scenarios_count"`
		DefaultStatus int    `json:"default_status"`
	}
	base := s.baseURL()
	items := make([]epItem, 0, len(endpoints))
	for _, e := range endpoints {
		scenarios, defaultStatus := countScenarios(e.Flow)
		items = append(items, epItem{
			ID:            e.ID,
			Name:          e.Name,
			Method:        e.Method,
			Path:          e.Path,
			CallableURL:   fmt.Sprintf("%s/public/%s%s", base, ws.Slug, e.Path),
			Scenarios:     scenarios,
			DefaultStatus: defaultStatus,
		})
	}
	b, _ := json.MarshalIndent(items, "", "  ")
	return successResult(string(b)), nil
}

func (s *Server) toolGetEndpoint(ctx context.Context, args json.RawMessage) (CallToolResult, error) {
	var input struct {
		Workspace  string `json:"workspace"`
		EndpointID string `json:"endpoint_id"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return errorResult("Invalid arguments: " + err.Error()), nil
	}
	ws, err := s.resolveWorkspace(ctx, input.Workspace)
	if err != nil {
		return errorResult(err.Error()), nil
	}
	ep, err := s.resolveEndpoint(ctx, ws.ID, input.EndpointID)
	if err != nil {
		return errorResult(err.Error()), nil
	}
	summary := summarizeEndpoint(ep, ws.Slug, s.baseURL())
	b, _ := json.MarshalIndent(summary, "", "  ")
	return successResult(string(b)), nil
}

func (s *Server) toolCreateEndpoint(ctx context.Context, args json.RawMessage) (CallToolResult, error) {
	var input struct {
		Workspace string `json:"workspace"`
		Method    string `json:"method"`
		Path      string `json:"path"`
		Name      string `json:"name"`
		Status    int    `json:"status"`
		Body      string `json:"body"`
		DelayMS   int    `json:"delay_ms"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return errorResult("Invalid arguments: " + err.Error()), nil
	}
	ws, err := s.resolveWorkspace(ctx, input.Workspace)
	if err != nil {
		return errorResult(err.Error()), nil
	}
	method := strings.ToUpper(strings.TrimSpace(input.Method))
	path := strings.TrimSpace(input.Path)
	if path == "" || !strings.HasPrefix(path, "/") {
		return errorResult("Path must start with '/'"), nil
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = fmt.Sprintf("%s %s", method, path)
	}
	status := input.Status
	if status == 0 {
		status = 200
	}
	body := strings.TrimSpace(input.Body)
	if body == "" {
		body = "{\n  \"message\": \"Hello from Mocklab\"\n}"
	} else if !json.Valid([]byte(body)) {
		return errorResult("body must be valid JSON"), nil
	}
	if input.DelayMS < 0 || input.DelayMS > 10000 {
		return errorResult("delay_ms must be between 0 and 10000"), nil
	}

	flow := domain.DefaultFlow()
	for i := range flow.Nodes {
		if flow.Nodes[i].Type == "response" {
			flow.Nodes[i].Config.Status = status
			flow.Nodes[i].Config.Body = body
			flow.Nodes[i].Config.DelayMS = input.DelayMS
			flow.Nodes[i].Config.ResponseMode = "static"
			flow.Nodes[i].Config.Headers = map[string]string{"Content-Type": "application/json"}
		}
	}

	endpoint := domain.Endpoint{
		ID:          usecase.ID(),
		WorkspaceID: ws.ID,
		Name:        name,
		Method:      method,
		Path:        path,
		Flow:        flow,
	}

	saved, err := s.service.SaveEndpoint(ctx, endpoint)
	if err != nil {
		return errorResult("Failed to save endpoint: " + err.Error()), nil
	}

	resultMap := map[string]any{
		"id":           saved.ID,
		"workspace_id": ws.ID,
		"name":         saved.Name,
		"method":       saved.Method,
		"path":         saved.Path,
		"callable_url": fmt.Sprintf("%s/public/%s%s", s.baseURL(), ws.Slug, saved.Path),
		"status":       status,
		"endpoint":     summarizeEndpoint(saved, ws.Slug, s.baseURL()),
		"message":      "Mock endpoint created successfully.",
	}
	b, _ := json.MarshalIndent(resultMap, "", "  ")
	return successResult(string(b)), nil
}

func (s *Server) toolAddScenario(ctx context.Context, args json.RawMessage) (CallToolResult, error) {
	var input struct {
		Workspace     string `json:"workspace"`
		EndpointID    string `json:"endpoint_id"`
		Name          string `json:"name"`
		Status        int    `json:"status"`
		Body          string `json:"body"`
		ConditionType string `json:"condition_type"`
		Field         string `json:"field"`
		Operator      string `json:"operator"`
		Value         string `json:"value"`
		DelayMS       int    `json:"delay_ms"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return errorResult("Invalid arguments: " + err.Error()), nil
	}
	ws, err := s.resolveWorkspace(ctx, input.Workspace)
	if err != nil {
		return errorResult(err.Error()), nil
	}
	ep, err := s.resolveEndpoint(ctx, ws.ID, input.EndpointID)
	if err != nil {
		return errorResult(err.Error()), nil
	}

	if input.Status < 200 || input.Status > 599 {
		return errorResult("status must be between 200 and 599"), nil
	}
	body := strings.TrimSpace(input.Body)
	if body == "" {
		body = "{}"
	} else if !json.Valid([]byte(body)) {
		return errorResult("body must be valid JSON"), nil
	}
	if input.DelayMS < 0 || input.DelayMS > 10000 {
		return errorResult("delay_ms must be between 0 and 10000"), nil
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = fmt.Sprintf("HTTP %d Scenario", input.Status)
	}
	condType := strings.ToLower(strings.TrimSpace(input.ConditionType))
	if !strings.Contains("header query path body", condType) {
		return errorResult("condition_type must be header, query, path, or body"), nil
	}
	field := strings.TrimSpace(input.Field)
	if field == "" {
		return errorResult("field cannot be empty"), nil
	}
	op := strings.ToLower(strings.TrimSpace(input.Operator))
	if !strings.Contains("equals not_equals contains exists empty", op) {
		return errorResult("operator must be equals, not_equals, contains, exists, or empty"), nil
	}

	newFlow, err := appendScenarioToFlow(ep.Flow, name, input.Status, body, input.DelayMS, condType, field, op, input.Value)
	if err != nil {
		return errorResult("Failed to construct scenario: " + err.Error()), nil
	}
	ep.Flow = newFlow

	saved, err := s.service.SaveEndpoint(ctx, ep)
	if err != nil {
		return errorResult("Failed to save endpoint scenario: " + err.Error()), nil
	}

	summary := summarizeEndpoint(saved, ws.Slug, s.baseURL())
	b, _ := json.MarshalIndent(map[string]any{
		"message":  fmt.Sprintf("Scenario '%s' added successfully.", name),
		"endpoint": summary,
	}, "", "  ")
	return successResult(string(b)), nil
}

func (s *Server) toolUpdateDefaultResponse(ctx context.Context, args json.RawMessage) (CallToolResult, error) {
	var input struct {
		Workspace  string  `json:"workspace"`
		EndpointID string  `json:"endpoint_id"`
		Status     *int    `json:"status"`
		Body       *string `json:"body"`
		DelayMS    *int    `json:"delay_ms"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return errorResult("Invalid arguments: " + err.Error()), nil
	}
	ws, err := s.resolveWorkspace(ctx, input.Workspace)
	if err != nil {
		return errorResult(err.Error()), nil
	}
	ep, err := s.resolveEndpoint(ctx, ws.ID, input.EndpointID)
	if err != nil {
		return errorResult(err.Error()), nil
	}

	fallbackID := findFallbackResponseID(ep.Flow)
	if fallbackID == "" {
		return errorResult("Could not locate default fallback response node"), nil
	}

	for i := range ep.Flow.Nodes {
		if ep.Flow.Nodes[i].ID == fallbackID {
			if input.Status != nil {
				if *input.Status < 200 || *input.Status > 599 {
					return errorResult("status must be 200–599"), nil
				}
				ep.Flow.Nodes[i].Config.Status = *input.Status
			}
			if input.Body != nil {
				if !json.Valid([]byte(*input.Body)) {
					return errorResult("body must be valid JSON"), nil
				}
				ep.Flow.Nodes[i].Config.Body = *input.Body
			}
			if input.DelayMS != nil {
				if *input.DelayMS < 0 || *input.DelayMS > 10000 {
					return errorResult("delay_ms must be 0–10000"), nil
				}
				ep.Flow.Nodes[i].Config.DelayMS = *input.DelayMS
			}
			break
		}
	}

	saved, err := s.service.SaveEndpoint(ctx, ep)
	if err != nil {
		return errorResult("Failed to update default response: " + err.Error()), nil
	}
	b, _ := json.MarshalIndent(map[string]any{
		"message":  "Default response updated successfully.",
		"endpoint": summarizeEndpoint(saved, ws.Slug, s.baseURL()),
	}, "", "  ")
	return successResult(string(b)), nil
}

func (s *Server) toolDeleteEndpoint(ctx context.Context, args json.RawMessage) (CallToolResult, error) {
	var input struct {
		Workspace  string `json:"workspace"`
		EndpointID string `json:"endpoint_id"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return errorResult("Invalid arguments: " + err.Error()), nil
	}
	ws, err := s.resolveWorkspace(ctx, input.Workspace)
	if err != nil {
		return errorResult(err.Error()), nil
	}
	if err := s.service.DeleteEndpoint(ctx, ws.ID, input.EndpointID); err != nil {
		return errorResult("Failed to delete endpoint: " + err.Error()), nil
	}
	return successResult(fmt.Sprintf("Endpoint %s deleted from workspace %s.", input.EndpointID, ws.Slug)), nil
}

func (s *Server) toolTestMock(ctx context.Context, args json.RawMessage) (CallToolResult, error) {
	var input struct {
		Workspace  string              `json:"workspace"`
		EndpointID string              `json:"endpoint_id"`
		Params     map[string]string   `json:"params"`
		Query      map[string][]string `json:"query"`
		Headers    map[string]string   `json:"headers"`
		Body       any                 `json:"body"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return errorResult("Invalid arguments: " + err.Error()), nil
	}
	ws, err := s.resolveWorkspace(ctx, input.Workspace)
	if err != nil {
		return errorResult(err.Error()), nil
	}
	ep, err := s.resolveEndpoint(ctx, ws.ID, input.EndpointID)
	if err != nil {
		return errorResult(err.Error()), nil
	}

	headers := http.Header{}
	for k, v := range input.Headers {
		headers.Set(k, v)
	}

	var bodyMap map[string]any
	rawBody := ""
	if input.Body != nil {
		if m, ok := input.Body.(map[string]any); ok {
			bodyMap = m
		} else if str, ok := input.Body.(string); ok {
			rawBody = str
			_ = json.Unmarshal([]byte(str), &bodyMap)
		} else {
			b, _ := json.Marshal(input.Body)
			rawBody = string(b)
			_ = json.Unmarshal(b, &bodyMap)
		}
	}

	req := usecase.Request{
		Method:      ep.Method,
		Path:        ep.Path,
		Headers:     headers,
		Query:       input.Query,
		Body:        bodyMap,
		RawBody:     rawBody,
		BodyPresent: input.Body != nil,
	}

	result, err := usecase.EvaluateWithEnvironment(ctx, ep.Flow, req, input.Params, nil)
	if err != nil {
		return errorResult("Evaluation error: " + err.Error()), nil
	}

	matchedLabel := "Default response"
	for _, n := range ep.Flow.Nodes {
		if n.Type == "response" && len(result.Trace) > 0 && result.Trace[len(result.Trace)-1] == n.ID {
			matchedLabel = n.Label
			break
		}
	}

	resMap := map[string]any{
		"status":           result.Status,
		"headers":          result.Headers,
		"body":             result.Body,
		"matched_scenario": matchedLabel,
		"trace":            result.Trace,
	}
	b, _ := json.MarshalIndent(resMap, "", "  ")
	return successResult(string(b)), nil
}

func (s *Server) toolGetWorkspaceSpec(ctx context.Context, args json.RawMessage) (CallToolResult, error) {
	var input struct {
		Workspace string `json:"workspace"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return errorResult("Invalid arguments: " + err.Error()), nil
	}
	ws, err := s.resolveWorkspace(ctx, input.Workspace)
	if err != nil {
		return errorResult(err.Error()), nil
	}
	endpoints, err := s.service.Endpoints(ctx, ws.ID)
	if err != nil {
		return errorResult("Failed to load endpoints: " + err.Error()), nil
	}

	type specOutput struct {
		Workspace struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			Slug          string `json:"slug"`
			BaseURL       string `json:"base_url"`
			PublicDocsURL string `json:"public_docs_url"`
		} `json:"workspace"`
		Endpoints []any `json:"endpoints"`
	}

	base := s.baseURL()
	out := specOutput{}
	out.Workspace.ID = ws.ID
	out.Workspace.Name = ws.Name
	out.Workspace.Slug = ws.Slug
	out.Workspace.BaseURL = fmt.Sprintf("%s/public/%s", base, ws.Slug)
	out.Workspace.PublicDocsURL = fmt.Sprintf("%s/share/%s", base, ws.Slug)

	for _, ep := range endpoints {
		out.Endpoints = append(out.Endpoints, summarizeEndpoint(ep, ws.Slug, base))
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	return successResult(string(b)), nil
}

// Helpers

func (s *Server) resolveWorkspace(ctx context.Context, identifier string) (domain.Workspace, error) {
	identifier = strings.TrimSpace(identifier)
	workspaces, err := s.service.Workspaces(ctx)
	if err != nil {
		return domain.Workspace{}, err
	}
	for _, w := range workspaces {
		if w.ID == identifier || w.Slug == identifier {
			return w, nil
		}
	}
	return domain.Workspace{}, fmt.Errorf("workspace '%s' not found", identifier)
}

func (s *Server) resolveEndpoint(ctx context.Context, wid, eid string) (domain.Endpoint, error) {
	endpoints, err := s.service.Endpoints(ctx, wid)
	if err != nil {
		return domain.Endpoint{}, err
	}
	for _, e := range endpoints {
		if e.ID == eid {
			return e, nil
		}
	}
	return domain.Endpoint{}, fmt.Errorf("endpoint '%s' not found in workspace", eid)
}

func countScenarios(f domain.Flow) (int, int) {
	scenarios := 0
	defaultStatus := 200
	for _, n := range f.Nodes {
		if n.Type == "condition" {
			scenarios++
		} else if n.Type == "response" {
			defaultStatus = n.Config.Status
		}
	}
	return scenarios + 1, defaultStatus
}

func findFallbackResponseID(f domain.Flow) string {
	nodeMap := make(map[string]domain.Node)
	for _, n := range f.Nodes {
		nodeMap[n.ID] = n
	}
	for _, e := range f.Edges {
		if e.Branch == "false" || (e.Branch == "" && e.From == "request") {
			if target, ok := nodeMap[e.To]; ok && target.Type == "response" {
				return target.ID
			}
		}
	}
	for _, n := range f.Nodes {
		if n.Type == "response" {
			return n.ID
		}
	}
	return ""
}

func appendScenarioToFlow(f domain.Flow, name string, status int, body string, delayMS int, condType, field, op, val string) (domain.Flow, error) {
	fallbackID := findFallbackResponseID(f)
	if fallbackID == "" {
		return f, errors.New("cannot find fallback response node in flow")
	}

	newCondID := usecase.ID()
	newRespID := usecase.ID()

	newResponseNode := domain.Node{
		ID:    newRespID,
		Type:  "response",
		Label: name,
		X:     820,
		Y:     float64(100 + len(f.Nodes)*120),
		Config: domain.Config{
			Status:       status,
			Body:         body,
			DelayMS:      delayMS,
			ResponseMode: "static",
			Headers:      map[string]string{"Content-Type": "application/json"},
		},
	}

	rule := domain.ConditionRule{
		Field:    field,
		Operator: op,
		Value:    val,
		Required: true,
	}

	newCondNode := domain.Node{
		ID:    newCondID,
		Type:  "condition",
		Label: name,
		X:     450,
		Y:     float64(100 + len(f.Nodes)*120),
		Config: domain.Config{
			ConditionType: condType,
			Match:         "all",
			Rules:         []domain.ConditionRule{rule},
			BodyMode:      "raw",
			RawFormat:     "json",
		},
	}

	// Find the incoming edge that currently lands on fallbackID with branch "" (from request) or "false" (from previous condition)
	var targetEdgeIndex = -1
	for i, edge := range f.Edges {
		if edge.To == fallbackID && (edge.Branch == "false" || edge.Branch == "") {
			targetEdgeIndex = i
			break
		}
	}
	if targetEdgeIndex == -1 {
		return f, errors.New("could not find unbranched or fallback connection to response")
	}

	f.Edges[targetEdgeIndex].To = newCondID

	f.Edges = append(f.Edges,
		domain.Edge{From: newCondID, To: newRespID, Branch: "true"},
		domain.Edge{From: newCondID, To: fallbackID, Branch: "false"},
	)

	f.Nodes = append(f.Nodes, newCondNode, newResponseNode)
	if err := f.Validate(); err != nil {
		return f, err
	}
	return f, nil
}

func summarizeEndpoint(ep domain.Endpoint, workspaceSlug, baseURL string) map[string]any {
	nodes := make(map[string]domain.Node)
	for _, n := range ep.Flow.Nodes {
		nodes[n.ID] = n
	}

	scenarios := []map[string]any{}
	priority := 1
	for _, n := range ep.Flow.Nodes {
		if n.Type == "condition" {
			// Find response node on true branch
			var trueTarget *domain.Node
			for _, edge := range ep.Flow.Edges {
				if edge.From == n.ID && edge.Branch == "true" {
					if target, ok := nodes[edge.To]; ok {
						trueTarget = &target
					}
					break
				}
			}
			sc := map[string]any{
				"priority":       priority,
				"name":           n.Label,
				"condition_type": n.Config.ConditionType,
				"rules":          n.Config.Rules,
			}
			if trueTarget != nil {
				sc["response_status"] = trueTarget.Config.Status
				sc["response_body"] = trueTarget.Config.Body
				sc["delay_ms"] = trueTarget.Config.DelayMS
			}
			scenarios = append(scenarios, sc)
			priority++
		}
	}

	fallbackID := findFallbackResponseID(ep.Flow)
	defaultResp := map[string]any{
		"status": 200,
		"body":   "{}",
	}
	if fb, ok := nodes[fallbackID]; ok {
		defaultResp["status"] = fb.Config.Status
		defaultResp["body"] = fb.Config.Body
		defaultResp["delay_ms"] = fb.Config.DelayMS
	}

	return map[string]any{
		"id":               ep.ID,
		"name":             ep.Name,
		"method":           ep.Method,
		"path":             ep.Path,
		"callable_url":     fmt.Sprintf("%s/public/%s%s", baseURL, workspaceSlug, ep.Path),
		"scenarios":        scenarios,
		"default_response": defaultResp,
	}
}

func successResult(text string) CallToolResult {
	return CallToolResult{
		Content: []ContentItem{
			{Type: "text", Text: text},
		},
		IsError: false,
	}
}

func errorResult(msg string) CallToolResult {
	return CallToolResult{
		Content: []ContentItem{
			{Type: "text", Text: msg},
		},
		IsError: true,
	}
}
