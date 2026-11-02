package domain

import "testing"

func TestFlowCannotOverrideCORSPolicy(t *testing.T) {
	for _, header := range []string{"Access-Control-Allow-Origin", "access-control-allow-credentials", "Access-Control-Allow-Headers"} {
		f := DefaultFlow()
		f.Nodes[1].Config.Headers[header] = "*"
		if f.Validate() == nil {
			t.Fatalf("flow accepted reserved header %s", header)
		}
	}
}
