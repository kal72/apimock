package config

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	Environment   string
	PublicBaseURL string
	AdminUsername string
	AdminPassword string
	CORSOrigins   []string
	CORSHeaders   []string
}

func Load() (Config, error) {
	c := Config{
		Environment:   env("APP_ENV", "development"),
		PublicBaseURL: strings.TrimRight(os.Getenv("PUBLIC_BASE_URL"), "/"),
		AdminUsername: os.Getenv("ADMIN_USERNAME"),
		AdminPassword: os.Getenv("ADMIN_PASSWORD"),
		CORSOrigins:   list(os.Getenv("MOCK_CORS_ORIGINS")),
		CORSHeaders:   list(env("MOCK_CORS_HEADERS", "Content-Type,Authorization,X-Api-Key")),
	}
	return c, c.Validate()
}

func (c Config) Validate() error {
	if c.Environment != "development" && c.Environment != "internal" && c.Environment != "production" {
		return errors.New("APP_ENV must be development, internal, or production")
	}
	if c.PublicBaseURL != "" {
		if !validOrigin(c.PublicBaseURL) {
			return errors.New("PUBLIC_BASE_URL must be an http(s) origin without a path, credentials, query, or fragment")
		}
	}
	if (c.AdminUsername == "") != (c.AdminPassword == "") {
		return errors.New("ADMIN_USERNAME and ADMIN_PASSWORD must be configured together")
	}
	if strings.ContainsAny(c.AdminUsername, ":\r\n") || len(c.AdminUsername) > 128 {
		return errors.New("ADMIN_USERNAME is invalid")
	}
	if c.AdminPassword != "" && (len(c.AdminPassword) < 16 || len(c.AdminPassword) > 1024 || strings.ContainsAny(c.AdminPassword, "\r\n")) {
		return errors.New("ADMIN_PASSWORD must contain 16–1024 bytes without line breaks")
	}
	if c.Environment == "production" && (c.AdminUsername == "" || !strings.HasPrefix(c.PublicBaseURL, "https://")) {
		return errors.New("production requires administrator credentials and an https PUBLIC_BASE_URL")
	}
	if c.Environment == "internal" && (c.AdminUsername == "" || c.PublicBaseURL == "") {
		return errors.New("internal deployments require administrator credentials and a PUBLIC_BASE_URL")
	}
	for _, origin := range c.CORSOrigins {
		if origin == "*" {
			if len(c.CORSOrigins) != 1 {
				return errors.New("MOCK_CORS_ORIGINS wildcard must be used alone")
			}
			continue
		}
		if !validOrigin(origin) {
			return fmt.Errorf("invalid MOCK_CORS_ORIGINS entry: %q", origin)
		}
	}
	for _, header := range c.CORSHeaders {
		if !ValidHeaderName(header) {
			return fmt.Errorf("invalid MOCK_CORS_HEADERS entry: %q", header)
		}
	}
	return nil
}

func (c Config) ManagedDeployment() bool {
	return c.Environment == "internal" || c.Environment == "production"
}

func (c Config) PublicHost() string {
	u, _ := url.Parse(c.PublicBaseURL)
	if u == nil {
		return ""
	}
	return u.Host
}
func (c Config) AllowsOrigin(origin string) bool {
	if origin == c.PublicBaseURL && origin != "" {
		return true
	}
	for _, allowed := range c.CORSOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}
func (c Config) AllowsHeader(header string) bool {
	for _, allowed := range c.CORSHeaders {
		if strings.EqualFold(http.CanonicalHeaderKey(header), allowed) {
			return true
		}
	}
	return false
}
func ValidHeaderName(name string) bool {
	if name == "" {
		return false
	}
	for _, ch := range name {
		if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", ch)) {
			return false
		}
	}
	return true
}
func validOrigin(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != "" && u.User == nil && u.Path == "" && u.RawQuery == "" && !u.ForceQuery && u.Fragment == "" && !strings.ContainsAny(raw, "\r\n\\ *")
}
func list(value string) []string {
	out := []string{}
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
