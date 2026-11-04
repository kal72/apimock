package domain

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type ResponseExpression struct {
	Source     string
	Argument   any
	Default    any
	HasDefault bool
}
type ResponseToken struct {
	Start, End int
	Expression ResponseExpression
}

var responseSource = regexp.MustCompile(`^request\.(path|query|headers|form|body)\.[A-Za-z0-9_.$-]+$`)

func ParseResponseExpression(text string) (ResponseExpression, error) {
	var e ResponseExpression
	source, fallback, hasDefault := strings.Cut(strings.TrimSpace(text), "| default:")
	source = strings.TrimSpace(source)
	e.Source = source
	e.HasDefault = hasDefault
	if hasDefault {
		decoder := json.NewDecoder(strings.NewReader(strings.TrimSpace(fallback)))
		decoder.UseNumber()
		if !json.Valid([]byte(strings.TrimSpace(fallback))) {
			return e, fmt.Errorf("default must be valid JSON")
		}
		if err := decoder.Decode(&e.Default); err != nil {
			return e, fmt.Errorf("default must be valid JSON")
		}
	}
	if responseSource.MatchString(source) || source == "request.body" || source == "request.raw_body" {
		return e, nil
	}
	if hasDefault {
		return e, fmt.Errorf("defaults are supported for request variables only")
	}
	if source == "generate.uuid" || source == "generate.timestamp" {
		return e, nil
	}
	name, args, ok := strings.Cut(source, ":")
	if !ok {
		return e, fmt.Errorf("unknown response variable: %s", source)
	}
	name = strings.TrimSpace(name)
	e.Source = name
	if err := json.Unmarshal([]byte(strings.TrimSpace(args)), &e.Argument); err != nil {
		return e, fmt.Errorf("generator arguments must be valid JSON")
	}
	values, ok := e.Argument.([]any)
	if !ok {
		return e, fmt.Errorf("generator arguments must be a JSON array")
	}
	switch name {
	case "generate.integer":
		if len(values) != 2 {
			return e, fmt.Errorf("integer requires [minimum, maximum]")
		}
		a, aok := values[0].(float64)
		b, bok := values[1].(float64)
		if !aok || !bok || a < -1000000000 || b > 1000000000 || a > b || a != float64(int64(a)) || b != float64(int64(b)) {
			return e, fmt.Errorf("integer bounds must be whole numbers between -1000000000 and 1000000000, minimum <= maximum")
		}
	case "generate.choice":
		if len(values) == 0 || len(values) > 100 {
			return e, fmt.Errorf("choice requires 1–100 values")
		}
	default:
		return e, fmt.Errorf("unknown response generator: %s", name)
	}
	return e, nil
}
func ResponseTokens(text string) ([]ResponseToken, error) {
	tokens := []ResponseToken{}
	for pos := 0; pos < len(text); {
		offset := strings.Index(text[pos:], "{{")
		if offset < 0 {
			if strings.Contains(text[pos:], "}}") {
				return nil, fmt.Errorf("unexpected closing template delimiter")
			}
			break
		}
		start := pos + offset
		depth := 0
		quoted := false
		escaped := false
		end := -1
		for i := start + 2; i < len(text); i++ {
			ch := text[i]
			if quoted {
				if escaped {
					escaped = false
				} else if ch == '\\' {
					escaped = true
				} else if ch == '"' {
					quoted = false
				}
				continue
			}
			if ch == '"' {
				quoted = true
				continue
			}
			if depth == 0 && ch == '}' && i+1 < len(text) && text[i+1] == '}' {
				end = i
				break
			}
			if ch == '{' || ch == '[' {
				depth++
			}
			if ch == '}' || ch == ']' {
				depth--
				if depth < 0 {
					break
				}
			}
		}
		if end < 0 {
			return nil, fmt.Errorf("unclosed response expression")
		}
		expression, err := ParseResponseExpression(text[start+2 : end])
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, ResponseToken{start, end + 2, expression})
		pos = end + 2
	}
	return tokens, nil
}
func ValidateResponseTemplate(c Config) error {
	if c.ResponseMode != "" && c.ResponseMode != "static" && c.ResponseMode != "dynamic" {
		return fmt.Errorf("response mode must be static or dynamic")
	}
	if c.ResponseMode != "dynamic" {
		return nil
	}
	if len(c.ResponseSeed) > 256 {
		return fmt.Errorf("response seed must be at most 256 bytes")
	}
	count := 0
	validate := func(text string) error {
		tokens, err := ResponseTokens(text)
		count += len(tokens)
		if count > 200 {
			return fmt.Errorf("response templates allow at most 200 expressions")
		}
		return err
	}
	var walk func(any, int) error
	walk = func(value any, depth int) error {
		if depth > 64 {
			return fmt.Errorf("dynamic JSON nesting must not exceed 64 levels")
		}
		switch v := value.(type) {
		case string:
			return validate(v)
		case []any:
			for _, item := range v {
				if err := walk(item, depth+1); err != nil {
					return err
				}
			}
		case map[string]any:
			for key, item := range v {
				if strings.Contains(key, "{{") || strings.Contains(key, "}}") {
					return fmt.Errorf("JSON object keys must be static")
				}
				if err := walk(item, depth+1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if c.Body != "" {
		var body any
		if err := json.Unmarshal([]byte(c.Body), &body); err != nil {
			return fmt.Errorf("response body must be valid JSON")
		}
		if err := walk(body, 0); err != nil {
			return err
		}
	}
	for _, value := range c.Headers {
		if err := validate(value); err != nil {
			return err
		}
	}
	return nil
}
func ValidResponseHeader(key, value string) bool { return validHeader(key, value) }
