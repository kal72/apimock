package httpadapter

import (
	"apimock/internal/config"
	"apimock/internal/domain"
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func bodyFlow(mode, format string) domain.Flow {
	f := domain.DefaultFlow()
	f.Nodes[1].Config.Body = `{"matched":true}`
	f.Nodes = append(f.Nodes, domain.Node{ID: "check", Type: "condition", Config: domain.Config{ConditionType: "body", Match: "all", BodyMode: mode, RawFormat: format, Rules: []domain.ConditionRule{{Field: "email", Operator: "exists", Required: true}, {Field: "role", Operator: "equals", Value: "admin", Required: true}}}}, domain.Node{ID: "invalid", Type: "response", Config: domain.Config{Status: 400, Body: `{"error":"required"}`}})
	if format == "text" {
		f.Nodes[2].Config.Rules = []domain.ConditionRule{{Field: "$raw", Operator: "contains", Value: "hello", Required: true}}
	}
	f.Edges = []domain.Edge{{From: "request", To: "check"}, {From: "check", To: "response", Branch: "true"}, {From: "check", To: "invalid", Branch: "false"}}
	return f
}
func TestPublicTypedBodyAndRequired(t *testing.T) {
	for _, tc := range []struct {
		name, mode, format, contentType, body string
		want                                  int
	}{
		{"JSON valid", "raw", "json", "application/json", `{"email":"a@example.com","role":"admin"}`, 200},
		{"JSON whitespace required", "raw", "json", "application/json", `{"email":"  ","role":"admin"}`, 400},
		{"JSON null required", "raw", "json", "application/json", `{"email":null,"role":"admin"}`, 400},
		{"Form valid", "form", "", "application/x-www-form-urlencoded", "email=a%40example.com&role=admin", 200},
		{"Form empty required", "form", "", "application/x-www-form-urlencoded", "email=&role=admin", 400},
		{"Form missing required", "form", "", "application/x-www-form-urlencoded", "role=admin", 400},
		{"format mismatch", "form", "", "application/json", `{"email":"a","role":"admin"}`, 400},
		{"Text valid", "raw", "text", "text/plain", "hello world", 200},
		{"Text blank", "raw", "text", "text/plain", "  ", 400},
		{"malformed JSON", "raw", "json", "application/json", `{"email":`, 400},
		{"trailing JSON", "raw", "json", "application/json", `{} {}`, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := configuredHandler(config.Config{})
			repo := h.Service.Repo.(*testRepository)
			repo.endpoints[0].Method = "POST"
			repo.endpoints[0].Flow = bodyFlow(tc.mode, tc.format)
			r := httptest.NewRequest("POST", "/public/demo/products/123", strings.NewReader(tc.body))
			r.Header.Set("Content-Type", tc.contentType)
			w := httptest.NewRecorder()
			h.Routes().ServeHTTP(w, r)
			if w.Code != tc.want {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}
func TestMultipartBodyFieldsAndBoundedRawBody(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	writer.WriteField("email", "a@example.com")
	writer.WriteField("role", "admin")
	part, _ := writer.CreateFormFile("file", "ignored.txt")
	part.Write([]byte("content"))
	writer.Close()
	h := configuredHandler(config.Config{})
	repo := h.Service.Repo.(*testRepository)
	repo.endpoints[0].Method = "POST"
	repo.endpoints[0].Flow = bodyFlow("form", "")
	r := httptest.NewRequest("POST", "/public/demo/products/123", &body)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	h.Routes().ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("multipart: %d %s", w.Code, w.Body.String())
	}
	r = httptest.NewRequest("POST", "/public/demo/products/123", strings.NewReader(strings.Repeat("x", (1<<20)+1)))
	r.Header.Set("Content-Type", "text/plain")
	w = httptest.NewRecorder()
	h.Routes().ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatalf("oversized request accepted: %d", w.Code)
	}
}
func TestPlaygroundFormAndRaw(t *testing.T) {
	for _, mode := range []string{"form", "text"} {
		format := ""
		raw := ""
		if mode == "text" {
			format = "text"
			raw = "hello"
		}
		flow := bodyFlow("form", format)
		if mode == "text" {
			flow = bodyFlow("raw", "text")
		}
		payload := map[string]any{"flow": flow, "body_mode": mode, "raw_body": raw, "form": map[string][]string{"email": {"a@example.com"}, "role": {"admin"}}}
		b, _ := json.Marshal(payload)
		w := httptest.NewRecorder()
		testHandler().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/test", bytes.NewReader(b)))
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"status":200`) {
			t.Fatalf("playground %s: %d %s", mode, w.Code, w.Body.String())
		}
	}
}
