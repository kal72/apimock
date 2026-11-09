package domain

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Environment is private to one user and can be selected in the Playground.
type Environment struct {
	ID        string                `json:"id"`
	Name      string                `json:"name"`
	Variables []EnvironmentVariable `json:"variables"`
}

type EnvironmentVariable struct {
	Key      string          `json:"key"`
	Kind     string          `json:"kind"`
	Value    json.RawMessage `json:"value,omitempty"`
	HasValue bool            `json:"has_value,omitempty"`
}

var environmentKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,63}$`)

func (e Environment) Validate() error {
	e.Name = strings.TrimSpace(e.Name)
	if e.Name == "" || len(e.Name) > 80 {
		return fmt.Errorf("environment name must be 1–80 characters")
	}
	return nil
}

func (v EnvironmentVariable) Validate() error {
	if !environmentKey.MatchString(v.Key) {
		return fmt.Errorf("variable key must start with a letter or underscore and contain only letters, numbers, or underscores")
	}
	if v.Kind != "string" && v.Kind != "number" && v.Kind != "boolean" && v.Kind != "secret" {
		return fmt.Errorf("variable kind must be string, number, boolean, or secret")
	}
	if len(v.Value) == 0 || !json.Valid(v.Value) {
		return fmt.Errorf("variable value must be valid JSON")
	}
	var value any
	if err := json.Unmarshal(v.Value, &value); err != nil {
		return fmt.Errorf("variable value must be valid JSON")
	}
	switch v.Kind {
	case "string", "secret":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("%s variable value must be a string", v.Kind)
		}
	case "number":
		if _, ok := value.(float64); !ok {
			return fmt.Errorf("number variable value must be a number")
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("boolean variable value must be true or false")
		}
	}
	return nil
}
