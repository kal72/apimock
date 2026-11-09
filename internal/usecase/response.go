package usecase

import (
	"apimock/internal/domain"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	mathrand "math/rand/v2"
	"strings"
	"time"
)

var ErrResponseRender = errors.New("mock response rendering failed")

func renderResponse(c domain.Config, r Request, params map[string]string) (string, map[string]string, error) {
	return renderResponseWithEnvironment(c, r, params, nil)
}
func renderResponseWithEnvironment(c domain.Config, r Request, params map[string]string, environment map[string]any) (string, map[string]string, error) {
	if c.ResponseMode != "dynamic" {
		return c.Body, c.Headers, nil
	}
	var seed [32]byte
	if c.ResponseSeed != "" {
		seed = sha256.Sum256([]byte(c.ResponseSeed))
	} else if _, err := rand.Read(seed[:]); err != nil {
		return "", nil, err
	}

	cache := map[string]any{}
	resolve := func(e domain.ResponseExpression) (any, error) {
		if strings.HasPrefix(e.Source, "generate.") {
			argument, _ := json.Marshal(e.Argument)
			key := e.Source + string(argument)
			expressionSeed := sha256.Sum256(append(seed[:], []byte(key)...))
			rng := mathrand.New(mathrand.NewChaCha8(expressionSeed))
			if value, ok := cache[key]; ok {
				return value, nil
			}
			var value any
			switch e.Source {
			case "generate.uuid":
				var b [16]byte
				binary.LittleEndian.PutUint64(b[:8], rng.Uint64())
				binary.LittleEndian.PutUint64(b[8:], rng.Uint64())
				b[6] = (b[6] & 15) | 64
				b[8] = (b[8] & 63) | 128
				value = fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
			case "generate.timestamp":
				value = time.Now().UTC().Format(time.RFC3339Nano)
			case "generate.integer":
				bounds := e.Argument.([]any)
				low, high := int64(bounds[0].(float64)), int64(bounds[1].(float64))
				value = low + rng.Int64N(high-low+1)
			case "generate.choice":
				choices := e.Argument.([]any)
				value = choices[rng.IntN(len(choices))]
			}
			cache[key] = value
			return value, nil
		}
		var value any
		var exists bool
		switch {
		case strings.HasPrefix(e.Source, "environment."):
			value, exists = environment[strings.TrimPrefix(e.Source, "environment.")]
		case e.Source == "request.raw_body":
			value = r.RawBody
			exists = r.BodyPresent || r.RawBody != ""
		case e.Source == "request.body":
			value = r.JSONBody
			if value == nil && r.Body != nil {
				value = r.Body
			}
			exists = (r.BodyFormat == "" || r.BodyFormat == "json") && (r.BodyPresent || r.JSONBody != nil || r.Body != nil)
		default:
			path := strings.TrimPrefix(e.Source, "request.")
			source, key, _ := strings.Cut(path, ".")
			switch source {
			case "path":
				value, exists = params[key]
			case "query":
				values, ok := r.Query[key]
				value, exists = first(values, ok)
			case "headers":
				value, exists = conditionValue(domain.Config{ConditionType: "header"}, key, r, params)
			case "form":
				values, ok := r.Form[key]
				value, exists = first(values, ok)
			case "body":
				root := r.JSONBody
				if root == nil {
					root = r.Body
				}
				value, exists = jsonField(root, key)
			}
		}
		if !exists {
			if e.HasDefault {
				return e.Default, nil
			}
			return nil, fmt.Errorf("missing variable %s; provide it in the request or select an environment, or add | default: JSON", e.Source)
		}
		return value, nil
	}
	renderText := func(text string, typed bool) (any, error) {
		tokens, err := domain.ResponseTokens(text)
		if err != nil {
			return nil, err
		}
		if typed && len(tokens) == 1 && tokens[0].Start == 0 && tokens[0].End == len(text) {
			return resolve(tokens[0].Expression)
		}
		var out strings.Builder
		pos := 0
		for _, token := range tokens {
			out.WriteString(text[pos:token.Start])
			value, err := resolve(token.Expression)
			if err != nil {
				return nil, err
			}
			out.WriteString(valueText(value))
			pos = token.End
			if out.Len() > 1<<20 {
				return nil, fmt.Errorf("rendered response exceeds 1 MiB")
			}
		}
		out.WriteString(text[pos:])
		return out.String(), nil
	}
	sizeBudget := 0
	var walk func(any) (any, error)
	walk = func(value any) (any, error) {
		switch v := value.(type) {
		case string:
			rendered, err := renderText(v, true)
			if err != nil {
				return nil, err
			}
			encoded, err := json.Marshal(rendered)
			if err != nil {
				return nil, err
			}
			sizeBudget += len(encoded)
			if sizeBudget > 1<<20 {
				return nil, fmt.Errorf("rendered response exceeds 1 MiB")
			}
			return rendered, nil
		case []any:
			for i, item := range v {
				rendered, err := walk(item)
				if err != nil {
					return nil, err
				}
				v[i] = rendered
			}
		case map[string]any:
			for key, item := range v {
				rendered, err := walk(item)
				if err != nil {
					return nil, err
				}
				v[key] = rendered
			}
		}
		return value, nil
	}
	body := ""
	if c.Body != "" {
		var root any
		decoder := json.NewDecoder(strings.NewReader(c.Body))
		decoder.UseNumber()
		if err := decoder.Decode(&root); err != nil {
			return "", nil, err
		}
		rendered, err := walk(root)
		if err != nil {
			return "", nil, err
		}
		data, err := json.Marshal(rendered)
		if err != nil {
			return "", nil, err
		}
		body = string(data)
	}
	headers := map[string]string{}
	size := len(body)
	for key, value := range c.Headers {
		rendered, err := renderText(value, false)
		if err != nil {
			return "", nil, err
		}
		text := rendered.(string)
		if !domain.ValidResponseHeader(key, text) {
			return "", nil, fmt.Errorf("rendered header %s is invalid or restricted", key)
		}
		headers[key] = text
		size += len(key) + len(text)
	}
	if size > 1<<20 {
		return "", nil, fmt.Errorf("rendered response exceeds 1 MiB")
	}
	return body, headers, nil
}
