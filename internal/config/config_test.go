package config

import "testing"

func TestProductionRequirements(t *testing.T) {
	valid := Config{Environment: "production", PublicBaseURL: "https://mock.example.com", AdminUsername: "admin", AdminPassword: "a-long-test-password", CORSOrigins: []string{"https://frontend.example.com"}, CORSHeaders: []string{"Authorization", "Content-Type"}}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*Config){
		"missing password":       func(c *Config) { c.AdminPassword = "" },
		"short password":         func(c *Config) { c.AdminPassword = "short" },
		"missing username":       func(c *Config) { c.AdminUsername = "" },
		"http public url":        func(c *Config) { c.PublicBaseURL = "http://mock.example.com" },
		"public url path":        func(c *Config) { c.PublicBaseURL = "https://mock.example.com/prefix" },
		"public url credentials": func(c *Config) { c.PublicBaseURL = "https://user:secret@mock.example.com" },
		"unknown environment":    func(c *Config) { c.Environment = "prodution" },
		"mixed wildcard":         func(c *Config) { c.CORSOrigins = []string{"*", "https://frontend.example.com"} },
		"origin path":            func(c *Config) { c.CORSOrigins = []string{"https://frontend.example.com/path"} },
		"invalid header":         func(c *Config) { c.CORSHeaders = []string{"X-Test\r\nInjected"} },
	} {
		t.Run(name, func(t *testing.T) {
			c := valid
			mutate(&c)
			if c.Validate() == nil {
				t.Fatal("expected invalid configuration")
			}
		})
	}
}

func TestLoadLocalDefaultsAndPublicOrigin(t *testing.T) {
	for _, key := range []string{"PUBLIC_BASE_URL", "ADMIN_USERNAME", "ADMIN_PASSWORD", "MOCK_CORS_ORIGINS"} {
		t.Setenv(key, "")
	}
	t.Setenv("APP_ENV", "development")
	t.Setenv("MOCK_CORS_HEADERS", "Content-Type,Authorization,X-Api-Key")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.PublicBaseURL != "" || len(c.CORSOrigins) != 0 || !c.AllowsHeader("authorization") {
		t.Fatalf("unexpected defaults: %+v", c)
	}
	t.Setenv("PUBLIC_BASE_URL", "https://mock.example.com/")
	t.Setenv("MOCK_CORS_ORIGINS", " https://frontend.example.com, https://other.example.com ")
	c, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.PublicBaseURL != "https://mock.example.com" || c.PublicHost() != "mock.example.com" || !c.AllowsOrigin("https://frontend.example.com") || c.AllowsOrigin("https://frontend.example.com.evil") {
		t.Fatal("public origin parsing or exact origin matching failed")
	}
}

func TestInternalDeploymentRequirements(t *testing.T) {
	valid := Config{Environment: "internal", PublicBaseURL: "http://apimock.internal:8080", AdminUsername: "admin", AdminPassword: "test-password-long-enough"}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	if !valid.ManagedDeployment() || valid.PublicHost() != "apimock.internal:8080" {
		t.Fatal("internal deployment must enforce its configured host")
	}
	for _, mutate := range []func(*Config){func(c *Config) { c.AdminUsername = ""; c.AdminPassword = "" }, func(c *Config) { c.PublicBaseURL = "" }, func(c *Config) { c.PublicBaseURL = "http://apimock.internal/public/demo" }} {
		c := valid
		mutate(&c)
		if c.Validate() == nil {
			t.Fatal("incomplete internal configuration was accepted")
		}
	}
}

func TestBootstrapPasswordMinimumEightCharacters(t *testing.T) {
	for _, password := range []string{"1234567", "密碼測試"} {
		if err := (Config{Environment: "development", AdminUsername: "admin", AdminPassword: password}).Validate(); err == nil {
			t.Fatal("short bootstrap password accepted")
		}
	}
	if err := (Config{Environment: "development", AdminUsername: "admin", AdminPassword: "12345678"}).Validate(); err != nil {
		t.Fatal(err)
	}
}
