package domain

import (
	"encoding/json"
	"testing"
)

func TestEnvironmentVariableValidate(t *testing.T) {
	tests := []struct {
		name    string
		v       EnvironmentVariable
		wantErr bool
	}{
		{
			name: "valid string",
			v: EnvironmentVariable{
				Key:   "api_url",
				Kind:  "string",
				Value: json.RawMessage(`"https://api.example.com"`),
			},
			wantErr: false,
		},
		{
			name: "valid secret",
			v: EnvironmentVariable{
				Key:   "api_secret",
				Kind:  "secret",
				Value: json.RawMessage(`"secret-123"`),
			},
			wantErr: false,
		},
		{
			name: "valid number integer",
			v: EnvironmentVariable{
				Key:   "port",
				Kind:  "number",
				Value: json.RawMessage(`8080`),
			},
			wantErr: false,
		},
		{
			name: "valid number float",
			v: EnvironmentVariable{
				Key:   "rate",
				Kind:  "number",
				Value: json.RawMessage(`3.14`),
			},
			wantErr: false,
		},
		{
			name: "valid boolean true",
			v: EnvironmentVariable{
				Key:   "enabled",
				Kind:  "boolean",
				Value: json.RawMessage(`true`),
			},
			wantErr: false,
		},
		{
			name: "valid boolean false",
			v: EnvironmentVariable{
				Key:   "enabled",
				Kind:  "boolean",
				Value: json.RawMessage(`false`),
			},
			wantErr: false,
		},
		{
			name: "invalid key with hyphen",
			v: EnvironmentVariable{
				Key:   "api-url",
				Kind:  "string",
				Value: json.RawMessage(`"test"`),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.v.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

