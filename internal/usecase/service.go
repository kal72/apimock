package usecase

import (
	"apimock/internal/domain"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

type Repository interface {
	Workspaces(context.Context) ([]domain.Workspace, error)
	SaveWorkspace(context.Context, domain.Workspace) error
	DeleteWorkspace(context.Context, string) error
	Endpoints(context.Context, string) ([]domain.Endpoint, error)
	SaveEndpoint(context.Context, domain.Endpoint) error
	DeleteEndpoint(context.Context, string, string) error
}
type EnvironmentRepository interface {
	Environments(context.Context, string) ([]domain.Environment, error)
	SaveEnvironment(context.Context, string, domain.Environment) error
	DeleteEnvironment(context.Context, string, string) error
	SaveEnvironmentVariable(context.Context, string, string, domain.EnvironmentVariable) error
	DeleteEnvironmentVariable(context.Context, string, string, string) error
	EnvironmentValues(context.Context, string, string) (map[string]any, error)
}
type Service struct{ Repo Repository }

func NewService(repo Repository) Service {
	return Service{Repo: repo}
}

func (s Service) Workspaces(ctx context.Context) ([]domain.Workspace, error) {
	return s.Repo.Workspaces(ctx)
}

func (s Service) DeleteWorkspace(ctx context.Context, id string) error {
	return s.Repo.DeleteWorkspace(ctx, id)
}

func (s Service) Endpoints(ctx context.Context, wid string) ([]domain.Endpoint, error) {
	return s.Repo.Endpoints(ctx, wid)
}

func (s Service) DeleteEndpoint(ctx context.Context, wid, eid string) error {
	return s.Repo.DeleteEndpoint(ctx, wid, eid)
}

func (s Service) Environments(ctx context.Context, userID string) ([]domain.Environment, error) {
	return s.EnvironmentRepository().Environments(ctx, userID)
}

func (s Service) DeleteEnvironment(ctx context.Context, userID, envID string) error {
	return s.EnvironmentRepository().DeleteEnvironment(ctx, userID, envID)
}

func (s Service) DeleteEnvironmentVariable(ctx context.Context, userID, envID, key string) error {
	return s.EnvironmentRepository().DeleteEnvironmentVariable(ctx, userID, envID, key)
}

func (s Service) EnvironmentValues(ctx context.Context, userID, envID string) (map[string]any, error) {
	return s.EnvironmentRepository().EnvironmentValues(ctx, userID, envID)
}

func ID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}
func (s Service) SaveWorkspace(ctx context.Context, w domain.Workspace) (domain.Workspace, error) {
	if err := w.Validate(); err != nil {
		return w, fmt.Errorf("%w: %s", domain.ErrValidation, err)
	}
	if w.ID == "" {
		w.ID = ID()
	}
	return w, s.Repo.SaveWorkspace(ctx, w)
}
func (s Service) SaveEndpoint(ctx context.Context, e domain.Endpoint) (domain.Endpoint, error) {
	if e.ID == "" && len(e.Flow.Nodes) == 0 && len(e.Flow.Edges) == 0 {
		e.Flow = domain.DefaultFlow()
	}
	if err := e.Validate(); err != nil {
		return e, fmt.Errorf("%w: %s", domain.ErrValidation, err)
	}
	if e.ID == "" {
		e.ID = ID()
	}
	return e, s.Repo.SaveEndpoint(ctx, e)
}
func (s Service) EnvironmentRepository() EnvironmentRepository { return s.Repo.(EnvironmentRepository) }
func (s Service) SaveEnvironment(ctx context.Context, userID string, item domain.Environment) (domain.Environment, error) {
	item.Name = strings.TrimSpace(item.Name)
	if err := item.Validate(); err != nil {
		return item, fmt.Errorf("%w: %s", domain.ErrValidation, err)
	}
	if item.ID == "" {
		item.ID = ID()
	}
	return item, s.EnvironmentRepository().SaveEnvironment(ctx, userID, item)
}
func (s Service) SaveEnvironmentVariable(ctx context.Context, userID, environmentID string, item domain.EnvironmentVariable) error {
	if err := item.Validate(); err != nil {
		return fmt.Errorf("%w: %s", domain.ErrValidation, err)
	}
	return s.EnvironmentRepository().SaveEnvironmentVariable(ctx, userID, environmentID, item)
}

type Request struct {
	BodyFormat   string
	RawBody      string
	Form         map[string][]string
	JSONBody     any
	BodyPresent  bool
	Method, Path string
	Headers      http.Header
	Query        map[string][]string
	Body         map[string]any
}
type Result struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
	Trace   []string          `json:"trace"`
}

func (s Service) Execute(ctx context.Context, slug string, r Request) (Result, error) {
	ws, err := s.Repo.Workspaces(ctx)
	if err != nil {
		return Result{}, err
	}
	wid := ""
	for _, w := range ws {
		if w.Slug == slug {
			wid = w.ID
			break
		}
	}
	if wid == "" {
		return Result{}, domain.ErrNotFound
	}
	endpoints, err := s.Repo.Endpoints(ctx, wid)
	if err != nil {
		return Result{}, err
	}
	// More static segments win, independent of creation order.
	sort.SliceStable(endpoints, func(i, j int) bool { return specificity(endpoints[i].Path) > specificity(endpoints[j].Path) })
	for _, e := range endpoints {
		if e.Method != r.Method {
			continue
		}
		params, ok := match(e.Path, r.Path)
		if !ok {
			continue
		}
		return Evaluate(ctx, e.Flow, r, params)
	}
	return Result{}, domain.ErrNotFound
}
func specificity(path string) int {
	score := 0
	for _, p := range strings.Split(path, "/") {
		if !strings.HasPrefix(p, ":") {
			score++
		}
	}
	return score
}
func match(pattern, path string) (map[string]string, bool) {
	a, b := strings.Split(pattern, "/"), strings.Split(path, "/")
	if len(a) != len(b) {
		return nil, false
	}
	p := map[string]string{}
	for i, v := range a {
		if strings.HasPrefix(v, ":") {
			if b[i] == "" {
				return nil, false
			}
			p[v[1:]] = b[i]
		} else if v != b[i] {
			return nil, false
		}
	}
	return p, true
}
func Evaluate(ctx context.Context, f domain.Flow, r Request, params map[string]string) (Result, error) {
	return EvaluateWithEnvironment(ctx, f, r, params, nil)
}
func EvaluateWithEnvironment(ctx context.Context, f domain.Flow, r Request, params map[string]string, environment map[string]any) (Result, error) {
	if err := f.Validate(); err != nil {
		return Result{}, fmt.Errorf("%w: %s", domain.ErrValidation, err)
	}
	nodes := map[string]domain.Node{}
	current := ""
	for _, n := range f.Nodes {
		nodes[n.ID] = n
		if n.Type == "request" {
			current = n.ID
		}
	}
	result := Result{Headers: map[string]string{}}
	for step := 0; step < len(f.Nodes); step++ {
		n := nodes[current]
		result.Trace = append(result.Trace, n.ID)
		branch := ""
		switch n.Type {
		case "condition":
			branch = fmt.Sprint(conditionMatches(n.Config, r, params))
		case "delay":
			timer := time.NewTimer(time.Duration(n.Config.DelayMS) * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return Result{}, ctx.Err()
			case <-timer.C:
			}
		case "headers":
			for k, v := range n.Config.Headers {
				result.Headers[k] = v
			}
		case "response":
			if n.Config.DelayMS > 0 {
				timer := time.NewTimer(time.Duration(n.Config.DelayMS) * time.Millisecond)
				select {
				case <-ctx.Done():
					timer.Stop()
					return Result{}, ctx.Err()
				case <-timer.C:
				}
			}
			result.Status = n.Config.Status
			body, headers, err := renderResponseWithEnvironment(n.Config, r, params, environment)
			if err != nil {
				return Result{}, fmt.Errorf("%w: %s", ErrResponseRender, err)
			}
			result.Body = body
			for k, v := range headers {
				result.Headers[k] = v
			}
			return result, nil
		}
		next := ""
		for _, e := range f.Edges {
			if e.From == current && e.Branch == branch {
				next = e.To
				break
			}
		}
		if next == "" {
			return Result{}, errors.New("flow has no response")
		}
		current = next
	}
	return Result{}, errors.New("execution step limit exceeded")
}
func field(key string, r Request, params map[string]string) (string, bool) {
	prefix, name, _ := strings.Cut(key, ".")
	switch prefix {
	case "params":
		v, ok := params[name]
		return v, ok
	case "query":
		v, ok := r.Query[name]
		if ok && len(v) > 0 {
			return v[0], true
		}
	case "headers":
		v, ok := r.Headers[http.CanonicalHeaderKey(name)]
		if ok && len(v) > 0 {
			return v[0], true
		}
	case "body":
		var value any = r.Body
		for _, part := range strings.Split(name, ".") {
			m, ok := value.(map[string]any)
			if !ok {
				return "", false
			}
			value, ok = m[part]
			if !ok {
				return "", false
			}
		}
		if v, ok := value.(string); ok {
			return v, true
		}
		b, err := json.Marshal(value)
		return string(b), err == nil
	}
	return "", false
}
