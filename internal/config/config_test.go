package config

import (
	"github.com/Crowley723/conduit/internal/authorization"
	"strings"
	"testing"
)

func TestValidateAuthorizationConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError bool
		errMsg    string
	}{
		{
			name: "valid authorization config",
			config: &Config{
				Authorization: AuthorizationConfig{
					GroupScopes: map[string][]string{
						"admin": {
							authorization.ScopeMTLSRequestCert,
							authorization.ScopeMTLSReadCert,
							authorization.ScopeMTLSApproveCert,
						},
					},
				},
			},
			wantError: false,
		},
		{
			name: "empty config applies defaults",
			config: &Config{
				Authorization: AuthorizationConfig{},
			},
			wantError: false,
		},
		{
			name: "invalid scope in group",
			config: &Config{
				Authorization: AuthorizationConfig{
					GroupScopes: map[string][]string{
						"admin": {
							authorization.ScopeMTLSRequestCert,
							"invalid:scope",
						},
					},
				},
			},
			wantError: true,
			errMsg:    "contains invalid scope",
		},
		{
			name: "group with no scopes",
			config: &Config{
				Authorization: AuthorizationConfig{
					GroupScopes: map[string][]string{
						"admin": {},
					},
				},
			},
			wantError: true,
			errMsg:    "has no scopes defined",
		},
		{
			name: "all valid scopes",
			config: &Config{
				Authorization: AuthorizationConfig{
					GroupScopes: map[string][]string{
						"admin": authorization.GetAllValidScopes(),
					},
				},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validateAuthorizationConfig()
			if tt.wantError {
				if err == nil {
					t.Errorf("validateAuthorizationConfig() expected error but got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateAuthorizationConfig() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateAuthorizationConfig() unexpected error = %v", err)
				}
			}
		})
	}
}
