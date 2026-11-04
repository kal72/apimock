package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestResponseTemplateValidation(t *testing.T) {
	for _, expression := range []string{"request.path.id", "request.body.items.0.id", "request.body | default: {\"nested\":{\"value\":0}}", "generate.uuid", "generate.timestamp", "generate.integer: [-2, 2]", "generate.choice: [\"pending\",false,null]"} {
		body, _ := json.Marshal(map[string]string{"value": "{{ " + expression + " }}"})
		if err := ValidateResponseTemplate(Config{ResponseMode: "dynamic", Body: string(body)}); err != nil {
			t.Fatalf("%s: %v", expression, err)
		}
	}
	for _, expression := range []string{"unknown", "generate.integer: [3, 1]", "generate.integer: [1.5, 3]", "generate.choice: []", "generate.uuid | default: 0", "request.body.id | default: nope"} {
		body, _ := json.Marshal("{{ " + expression + " }}")
		if ValidateResponseTemplate(Config{ResponseMode: "dynamic", Body: string(body)}) == nil {
			t.Fatalf("accepted %s", expression)
		}
	}
	for _, body := range []string{`"{{ request.body.id"`, `"unexpected }}"`, `{"{{ request.body.key }}":1}`} {
		if ValidateResponseTemplate(Config{ResponseMode: "dynamic", Body: body}) == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	if ValidateResponseTemplate(Config{ResponseMode: "dynamic", Body: `{}`, ResponseSeed: strings.Repeat("x", 257)}) == nil {
		t.Fatal("seed limit missing")
	}
	if ValidateResponseTemplate(Config{ResponseMode: "script", Body: `{}`}) == nil {
		t.Fatal("invalid mode accepted")
	}
}
