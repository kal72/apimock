package handler

import (
	"apimock/internal/config"
	"apimock/internal/domain"
	"apimock/internal/usecase"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type memoryIdentityRepo struct {
	mu       sync.RWMutex
	users    map[string]domain.User
	sessions map[string]string // token_hash -> user_id
	apiKeys  map[string]domain.APIKey
}

func newMemoryIdentityRepo() *memoryIdentityRepo {
	return &memoryIdentityRepo{
		users:    make(map[string]domain.User),
		sessions: make(map[string]string),
		apiKeys:  make(map[string]domain.APIKey),
	}
}

func (m *memoryIdentityRepo) HasUsers(context.Context) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.users) > 0, nil
}
func (m *memoryIdentityRepo) CreateFirstUser(ctx context.Context, u domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[u.ID] = u
	return nil
}
func (m *memoryIdentityRepo) Users(context.Context) ([]domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]domain.User, 0, len(m.users))
	for _, u := range m.users {
		out = append(out, u)
	}
	return out, nil
}
func (m *memoryIdentityRepo) UserByUsername(ctx context.Context, name string) (domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, u := range m.users {
		if u.Username == name {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrNotFound
}
func (m *memoryIdentityRepo) SaveUser(ctx context.Context, u domain.User, actor string, create bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[u.ID] = u
	return nil
}
func (m *memoryIdentityRepo) DeleteUser(ctx context.Context, id, actor string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.users, id)
	return nil
}
func (m *memoryIdentityRepo) CreateSession(ctx context.Context, hash, id, passwordHash string, expires time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[hash] = id
	return nil
}
func (m *memoryIdentityRepo) SessionUser(ctx context.Context, hash string) (domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	uid, ok := m.sessions[hash]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	u, ok := m.users[uid]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}
func (m *memoryIdentityRepo) DeleteSession(ctx context.Context, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, hash)
	return nil
}
func (m *memoryIdentityRepo) ChangePassword(ctx context.Context, id, old, next string) error {
	return nil
}

func (m *memoryIdentityRepo) APIKeys(ctx context.Context, userID string) ([]domain.APIKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []domain.APIKey
	for _, k := range m.apiKeys {
		if k.UserID == userID {
			out = append(out, k)
		}
	}
	return out, nil
}

func (m *memoryIdentityRepo) CreateAPIKey(ctx context.Context, k domain.APIKey) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.apiKeys[k.ID] = k
	return nil
}

func (m *memoryIdentityRepo) DeleteAPIKey(ctx context.Context, userID, keyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k, ok := m.apiKeys[keyID]
	if !ok || k.UserID != userID {
		return domain.ErrNotFound
	}
	delete(m.apiKeys, keyID)
	return nil
}

func (m *memoryIdentityRepo) AuthenticateAPIKey(ctx context.Context, hash string) (domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, k := range m.apiKeys {
		if k.KeyHash == hash {
			u, ok := m.users[k.UserID]
			if ok && u.Active {
				return u, nil
			}
		}
	}
	return domain.User{}, usecase.ErrUnauthorized
}

func TestAPIKeyEndpoints(t *testing.T) {
	repo := newMemoryIdentityRepo()
	user := domain.User{
		ID:       "user-1",
		Username: "admin",
		Name:     "Admin User",
		Role:     "admin",
		Active:   true,
	}
	repo.users[user.ID] = user

	token := "1234567890123456789012345678901234567890123" // 43 characters
	repo.sessions[usecase.TokenHash(token)] = user.ID

	auth := usecase.AuthService{Repo: repo}
	h := Handler{
		Auth:    &Identity{Service: auth},
		Config:  config.Config{Environment: "development"},
		Service: usecase.NewService(&testRepository{}),
		Assets:  http.NotFoundHandler(),
	}
	routes := h.Routes()

	// 1. List API keys (initially empty)
	req1 := httptest.NewRequest("GET", "/api/auth/api-keys", nil)
	req1.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	rec1 := httptest.NewRecorder()
	routes.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec1.Code, rec1.Body.String())
	}
	var keys []domain.APIKey
	if err := json.Unmarshal(rec1.Body.Bytes(), &keys); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("expected 0 keys, got %d", len(keys))
	}

	// 2. Create API key
	createPayload := `{"name":"Cursor Mac"}`
	req2 := httptest.NewRequest("POST", "/api/auth/api-keys", strings.NewReader(createPayload))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-CSRF-Token", usecase.CSRFToken(token))
	req2.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	rec2 := httptest.NewRecorder()
	routes.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}
	var created domain.GeneratedAPIKey
	if err := json.Unmarshal(rec2.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to parse generated key: %v", err)
	}
	if created.ID == "" || !strings.HasPrefix(created.RawKey, "mlk_") {
		t.Fatalf("invalid generated key: %+v", created)
	}
	if created.Name != "Cursor Mac" {
		t.Fatalf("expected name 'Cursor Mac', got %s", created.Name)
	}

	// 3. Test MCP authentication with the generated raw key
	mcpReq := httptest.NewRequest("GET", "/api/mcp/sse", nil)
	mcpReq.Header.Set("Authorization", "Bearer "+created.RawKey)
	mcpRec := newInMemorySSERecorder()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mcpReq = mcpReq.WithContext(ctx)

	var sseWg sync.WaitGroup
	sseWg.Add(1)
	go func() {
		defer sseWg.Done()
		routes.ServeHTTP(mcpRec, mcpReq)
	}()

	for i := 0; i < 50; i++ {
		if strings.Contains(mcpRec.String(), "event: endpoint") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !strings.Contains(mcpRec.String(), "event: endpoint") {
		t.Fatal("expected endpoint event from MCP SSE")
	}

	cancel()
	sseWg.Wait()

	// 4. Delete API key
	delReq := httptest.NewRequest("DELETE", "/api/auth/api-keys/"+created.ID, nil)
	delReq.Header.Set("X-CSRF-Token", usecase.CSRFToken(token))
	delReq.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	delRec := httptest.NewRecorder()
	routes.ServeHTTP(delRec, delReq)

	if delRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", delRec.Code, delRec.Body.String())
	}

	// 5. Verify deleted from list
	req3 := httptest.NewRequest("GET", "/api/auth/api-keys", nil)
	req3.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	rec3 := httptest.NewRecorder()
	routes.ServeHTTP(rec3, req3)

	var keysAfter []domain.APIKey
	_ = json.Unmarshal(rec3.Body.Bytes(), &keysAfter)
	if len(keysAfter) != 0 {
		t.Fatalf("expected 0 keys after deletion, got %d", len(keysAfter))
	}
}
