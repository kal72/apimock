package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("already exists")
var ErrValidation = errors.New("validation failed")
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Workspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}
type Endpoint struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	Flow        Flow   `json:"flow"`
}
type Flow struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}
type Node struct {
	ID     string  `json:"id"`
	Type   string  `json:"type"`
	Label  string  `json:"label"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Config Config  `json:"config"`
}
type Config struct {
	ConditionType string            `json:"condition_type,omitempty"`
	Match         string            `json:"match,omitempty"`
	BodyMode      string            `json:"body_mode,omitempty"`
	RawFormat     string            `json:"raw_format,omitempty"`
	Rules         []ConditionRule   `json:"rules,omitempty"`
	Example       *RequestExample   `json:"example,omitempty"`
	Field         string            `json:"field,omitempty"`
	Operator      string            `json:"operator,omitempty"`
	Value         string            `json:"value,omitempty"`
	Status        int               `json:"status,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	Body          string            `json:"body,omitempty"`
	DelayMS       int               `json:"delay_ms,omitempty"`
}

type RequestExample struct {
	BodyMode string              `json:"body_mode,omitempty"`
	RawBody  string              `json:"raw_body,omitempty"`
	Form     map[string][]string `json:"form,omitempty"`
	Params   map[string]string   `json:"params"`
	Query    map[string][]string `json:"query"`
	Headers  map[string]string   `json:"headers"`
	Body     map[string]any      `json:"body"`
}
type Edge struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Branch string `json:"branch,omitempty"`
}

func (w Workspace) Validate() error {
	if strings.TrimSpace(w.Name) == "" || !slugPattern.MatchString(w.Slug) || len(w.Slug) > 60 {
		return errors.New("workspace requires a name and a slug using lowercase letters, numbers, or hyphens (maximum 60 characters)")
	}
	return nil
}
func (e Endpoint) Validate() error {
	switch e.Method {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
	default:
		return errors.New("invalid HTTP method")
	}
	if e.Name == "" || !strings.HasPrefix(e.Path, "/") || strings.ContainsAny(e.Path, "?# ") {
		return errors.New("name is required and path must start with / without query, fragment, or spaces")
	}
	seen := map[string]bool{}
	for _, s := range strings.Split(e.Path, "/") {
		if strings.HasPrefix(s, ":") {
			if len(s) == 1 || seen[s] {
				return errors.New("invalid path parameter")
			}
			seen[s] = true
		}
	}
	for _, node := range e.Flow.Nodes {
		if node.Type == "condition" && node.Config.ConditionType == "path" {
			for _, rule := range node.Config.Rules {
				if !seen[":"+rule.Field] {
					return fmt.Errorf("path parameter %s is not available in the route", rule.Field)
				}
			}
		}
	}
	return e.Flow.Validate()
}
func (f Flow) Validate() error {
	if len(f.Nodes) == 0 || len(f.Nodes) > 100 {
		return errors.New("flow membutuhkan 1–100 node")
	}
	nodes := map[string]Node{}
	start := ""
	requests := 0
	for _, n := range f.Nodes {
		if n.ID == "" {
			return errors.New("node ID is required")
		}
		if _, ok := nodes[n.ID]; ok {
			return errors.New("node IDs must be unique")
		}
		nodes[n.ID] = n
		switch n.Type {
		case "request":
			requests++
			start = n.ID
		case "condition":
			if err := ValidateCondition(n.Config); err != nil {
				return err
			}
		case "headers":
		case "delay":
			if n.Config.DelayMS < 0 || n.Config.DelayMS > 10000 {
				return errors.New("delay maksimal 10000 ms")
			}
		case "response":
			if n.Config.DelayMS < 0 || n.Config.DelayMS > 10000 {
				return errors.New("response delay maksimal 10000 ms")
			}
			if n.Config.Status < 200 || n.Config.Status > 599 {
				return errors.New("response status must be 200–599")
			}
			if n.Config.Body != "" && !json.Valid([]byte(n.Config.Body)) {
				return errors.New("response body must be valid JSON")
			}
		default:
			return fmt.Errorf("unknown node type: %s", n.Type)
		}
		for k, v := range n.Config.Headers {
			if !validHeader(k, v) {
				return errors.New("invalid or restricted header")
			}
		}
	}
	if requests != 1 {
		return errors.New("flow must have exactly one Request")
	}
	outgoing := map[string][]Edge{}
	for _, e := range f.Edges {
		if _, ok := nodes[e.From]; !ok {
			return errors.New("connection source not found")
		}
		if _, ok := nodes[e.To]; !ok {
			return errors.New("connection destination not found")
		}
		outgoing[e.From] = append(outgoing[e.From], e)
	}
	for id, n := range nodes {
		edges := outgoing[id]
		switch n.Type {
		case "response":
			if len(edges) != 0 {
				return errors.New("Response cannot have outgoing connections")
			}
		case "condition":
			if len(edges) != 2 || !((edges[0].Branch == "true" && edges[1].Branch == "false") || (edges[0].Branch == "false" && edges[1].Branch == "true")) {
				return errors.New("Condition must have true and false branches")
			}
		default:
			if len(edges) != 1 || edges[0].Branch != "" {
				return errors.New("node must have one unbranched connection")
			}
		}
	}
	state := map[string]int{}
	var visit func(string) error
	visit = func(id string) error {
		if state[id] == 1 {
			return errors.New("flow cannot contain cycles")
		}
		if state[id] == 2 {
			return nil
		}
		state[id] = 1
		for _, e := range outgoing[id] {
			if err := visit(e.To); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}
	if err := visit(start); err != nil {
		return err
	}
	if len(state) != len(nodes) {
		return errors.New("all nodes must be reachable from Request")
	}
	return nil
}
func validHeader(k, v string) bool {
	if strings.HasPrefix(strings.ToLower(k), "access-control-") {
		return false
	}
	if k == "" || strings.ContainsAny(v, "\r\n") {
		return false
	}
	for _, c := range k {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", c)) {
			return false
		}
	}
	switch strings.ToLower(k) {
	case "content-length", "transfer-encoding", "connection", "set-cookie":
		return false
	}
	return true
}
func DefaultFlow() Flow {
	return Flow{Nodes: []Node{{ID: "request", Type: "request", Label: "Incoming request", X: 80, Y: 150}, {ID: "response", Type: "response", Label: "Success response", X: 440, Y: 150, Config: Config{Status: 200, Headers: map[string]string{"Content-Type": "application/json"}, Body: "{\n  \"message\": \"Hello from Mocklab\"\n}"}}}, Edges: []Edge{{From: "request", To: "response"}}}
}
