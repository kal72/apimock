package handler

import (
	"apimock/internal/config"
	"apimock/internal/domain"
	"apimock/internal/usecase"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type mockEnvRepo struct {
	testRepository
	environments []domain.Environment
	variables    map[string]map[string]domain.EnvironmentVariable
}

func (m *mockEnvRepo) HasUsers(context.Context) (bool, error) { return true, nil }
func (m *mockEnvRepo) CreateFirstUser(context.Context, domain.User) error { return nil }
func (m *mockEnvRepo) Users(context.Context) ([]domain.User, error) { return nil, nil }
func (m *mockEnvRepo) UserByUsername(context.Context, string) (domain.User, error) {
	return domain.User{ID: "u1", Username: "admin", Role: "admin"}, nil
}
func (m *mockEnvRepo) SaveUser(context.Context, domain.User, string, bool) error { return nil }
func (m *mockEnvRepo) DeleteUser(context.Context, string, string) error { return nil }
func (m *mockEnvRepo) CreateSession(context.Context, string, string, string, time.Time) error { return nil }
func (m *mockEnvRepo) SessionUser(context.Context, string) (domain.User, error) {
	return domain.User{ID: "u1", Username: "admin", Role: "admin"}, nil
}
func (m *mockEnvRepo) DeleteSession(context.Context, string) error { return nil }
func (m *mockEnvRepo) ChangePassword(context.Context, string, string, string) error { return nil }
func (m *mockEnvRepo) APIKeys(context.Context, string) ([]domain.APIKey, error) { return nil, nil }
func (m *mockEnvRepo) CreateAPIKey(context.Context, domain.APIKey) error { return nil }
func (m *mockEnvRepo) DeleteAPIKey(context.Context, string, string) error { return nil }
func (m *mockEnvRepo) AuthenticateAPIKey(context.Context, string) (domain.User, error) { return domain.User{}, usecase.ErrUnauthorized }

func (m *mockEnvRepo) Environments(_ context.Context, userID string) ([]domain.Environment, error) {
	return m.environments, nil
}
func (m *mockEnvRepo) SaveEnvironment(_ context.Context, userID string, env domain.Environment) error {
	m.environments = append(m.environments, env)
	return nil
}
func (m *mockEnvRepo) DeleteEnvironment(_ context.Context, userID, id string) error {
	return nil
}
func (m *mockEnvRepo) SaveEnvironmentVariable(_ context.Context, userID, envID string, variable domain.EnvironmentVariable) error {
	if m.variables == nil {
		m.variables = make(map[string]map[string]domain.EnvironmentVariable)
	}
	if m.variables[envID] == nil {
		m.variables[envID] = make(map[string]domain.EnvironmentVariable)
	}
	m.variables[envID][variable.Key] = variable
	return nil
}
func (m *mockEnvRepo) DeleteEnvironmentVariable(_ context.Context, userID, envID, key string) error {
	if m.variables != nil && m.variables[envID] != nil {
		delete(m.variables[envID], key)
	}
	return nil
}
func (m *mockEnvRepo) EnvironmentValues(_ context.Context, userID, envID string) (map[string]any, error) {
	return nil, nil
}

func TestEnvironmentRouteVariables(t *testing.T) {
	repo := &mockEnvRepo{
		environments: []domain.Environment{
			{ID: "env123", Name: "Development"},
		},
	}
	authSvc := usecase.AuthService{Repo: repo}
	svc := usecase.Service{Repo: repo}
	h := Handler{
		Config:  config.Config{},
		Service: svc,
		Auth:    &Identity{Service: authSvc},
		Assets:  http.NotFoundHandler(),
	}

	router := h.Routes()

	rawToken := "1234567890123456789012345678901234567890123"
	csrf := usecase.CSRFToken(rawToken)

	// Put variable
	body, _ := json.Marshal(map[string]any{
		"kind":  "string",
		"value": "https://api.example.com",
	})
	req := httptest.NewRequest("PUT", "/api/environments/env123/variables/api_base_url", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrf)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: rawToken})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	t.Logf("Response code: %d, body: %s", w.Code, w.Body.String())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	savedVar, ok := repo.variables["env123"]["api_base_url"]
	if !ok {
		t.Fatalf("variable was not saved in mock repo")
	}
	t.Logf("Saved variable: %+v, raw value: %s", savedVar, string(savedVar.Value))
}
