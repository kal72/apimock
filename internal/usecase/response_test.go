package usecase

import (
	"apimock/internal/domain"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func dynamicFlow(body string) domain.Flow {
	f := domain.DefaultFlow()
	f.Nodes[1].Config.ResponseMode = "dynamic"
	f.Nodes[1].Config.Body = body
	return f
}
func TestDynamicResponseTypesDefaultsAndEscaping(t *testing.T) {
	f := dynamicFlow(`{"id":"{{ request.path.id }}","count":"{{ request.body.count }}","active":"{{ request.body.active }}","object":"{{ request.body.object }}","items":"{{ request.body.items }}","null":"{{ request.body.nil | default: 12 }}","default":"{{ request.body.missing | default: {\"name\":\"Guest\"} }}","message":"Hello {{ request.body.name }}","large":9007199254740993}`)
	request := Request{JSONBody: map[string]any{"count": json.Number("0"), "active": false, "object": map[string]any{"id": "x"}, "items": []any{1, 2}, "nil": nil, "name": "Alice\"\n{{ generate.uuid }}"}, BodyPresent: true, BodyFormat: "json"}
	result, err := Evaluate(context.Background(), f, request, map[string]string{"id": "123"})
	if err != nil {
		t.Fatal(err)
	}
	var data map[string]any
	d := json.NewDecoder(strings.NewReader(result.Body))
	d.UseNumber()
	if err = d.Decode(&data); err != nil {
		t.Fatal(err)
	}
	if data["id"] != "123" || data["count"] != json.Number("0") || data["active"] != false || data["null"] != nil || data["large"] != json.Number("9007199254740993") {
		t.Fatalf("lost types: %s", result.Body)
	}
	if data["message"] != "Hello Alice\"\n{{ generate.uuid }}" {
		t.Fatalf("unsafe escaping or recursive evaluation: %s", result.Body)
	}
	if data["default"].(map[string]any)["name"] != "Guest" || len(data["items"].([]any)) != 2 {
		t.Fatal(result.Body)
	}
}
func TestStaticResponsesRemainByteIdentical(t *testing.T) {
	f := domain.DefaultFlow()
	body := "{\n  \"literal\": \"{{ unknown.execute }}\"\n}"
	f.Nodes[1].Config.Body = body
	f.Nodes[1].Config.Headers["X-Literal"] = "{{ unknown }}"
	for _, mode := range []string{"", "static"} {
		f.Nodes[1].Config.ResponseMode = mode
		result, err := Evaluate(context.Background(), f, Request{}, nil)
		if err != nil || result.Body != body || result.Headers["X-Literal"] != "{{ unknown }}" {
			t.Fatalf("changed static response: %+v %v", result, err)
		}
	}
}
func TestDynamicGeneratorsAreSeededCachedAndOrderIndependent(t *testing.T) {
	f := dynamicFlow(`{"id":"{{ generate.uuid }}","again":"{{ generate.uuid }}","number":"{{ generate.integer: [1, 3] }}","choice":"{{ generate.choice: [\"pending\",\"paid\"] }}","now":"{{ generate.timestamp }}"}`)
	f.Nodes[1].Config.ResponseSeed = "reproducible"
	f.Nodes[1].Config.Headers["X-Request-ID"] = "{{ generate.uuid }}"
	var previous string
	for i := 0; i < 20; i++ {
		result, err := Evaluate(context.Background(), f, Request{}, nil)
		if err != nil {
			t.Fatal(err)
		}
		var data map[string]any
		json.Unmarshal([]byte(result.Body), &data)
		id := data["id"].(string)
		if len(id) != 36 || id[14] != '4' || id != data["again"] || id != result.Headers["X-Request-ID"] {
			t.Fatal(result.Body)
		}
		if previous != "" && id != previous {
			t.Fatal("seeded UUID changed")
		}
		previous = id
		if data["number"].(float64) < 1 || data["number"].(float64) > 3 {
			t.Fatal(result.Body)
		}
	}
	f.Nodes[1].Config.ResponseSeed = ""
	a, _ := Evaluate(context.Background(), f, Request{}, nil)
	b, _ := Evaluate(context.Background(), f, Request{}, nil)
	if a.Headers["X-Request-ID"] == b.Headers["X-Request-ID"] {
		t.Fatal("unseeded UUID repeated")
	}
}
func TestDynamicMissingVariablesAndUnsafeHeadersFail(t *testing.T) {
	f := dynamicFlow(`{"name":"{{ request.body.name }}"}`)
	_, err := Evaluate(context.Background(), f, Request{}, nil)
	if !errors.Is(err, ErrResponseRender) || !strings.Contains(err.Error(), "missing variable") {
		t.Fatal(err)
	}
	f = dynamicFlow(`{}`)
	f.Nodes[1].Config.Headers["X-Name"] = "{{ request.headers.X-Name }}"
	_, err = Evaluate(context.Background(), f, Request{Headers: http.Header{"X-Name": {"bad\r\nInjected: yes"}}}, nil)
	if !errors.Is(err, ErrResponseRender) {
		t.Fatalf("header injection accepted: %v", err)
	}
	f = dynamicFlow(`{"a":"{{ request.body.large }}","b":"{{ request.body.large }}"}`)
	_, err = Evaluate(context.Background(), f, Request{Body: map[string]any{"large": strings.Repeat("x", 600000)}}, nil)
	if !errors.Is(err, ErrResponseRender) {
		t.Fatal("oversized rendering accepted")
	}
	f = dynamicFlow(`"{{ request.body | default: true }}"`)
	result, err := Evaluate(context.Background(), f, Request{}, nil)
	if err != nil || result.Body != "true" {
		t.Fatalf("missing root default: %+v %v", result, err)
	}
	f = dynamicFlow(`"{{ request.query.tags }}"`)
	result, err = Evaluate(context.Background(), f, Request{Query: map[string][]string{"tags": {"first", "second"}}}, nil)
	if err != nil || result.Body != `"first"` {
		t.Fatalf("query source: %+v %v", result, err)
	}
	f = dynamicFlow(`"{{ request.form.name }}"`)
	result, err = Evaluate(context.Background(), f, Request{Form: map[string][]string{"name": {"Form user"}}, BodyFormat: "form"}, nil)
	if err != nil || result.Body != `"Form user"` {
		t.Fatal(result, err)
	}
}
