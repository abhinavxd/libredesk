package main

import (
	"os"
	"testing"

	oidcmodels "github.com/abhinavxd/libredesk/internal/oidc/models"
	"github.com/knadh/koanf/v2"
)

func TestIsLocalLoginEnabled(t *testing.T) {
	tests := []struct {
		name       string
		configured *bool
		providers  []oidcmodels.OIDC
		want       bool
	}{
		{name: "defaults to enabled", want: true},
		{name: "remains enabled when configured", configured: boolPointer(true), want: true},
		{name: "remains enabled without an enabled OIDC provider", configured: boolPointer(false), providers: []oidcmodels.OIDC{{Enabled: false}}, want: true},
		{name: "is disabled with an enabled OIDC provider", configured: boolPointer(false), providers: []oidcmodels.OIDC{{Enabled: true}}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ko.Delete(localLoginEnabledKey)
			if tt.configured != nil {
				if err := ko.Set(localLoginEnabledKey, *tt.configured); err != nil {
					t.Fatalf("set config: %v", err)
				}
			}
			t.Cleanup(func() { ko.Delete(localLoginEnabledKey) })

			if got := isLocalLoginEnabled(tt.providers); got != tt.want {
				t.Fatalf("isLocalLoginEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientIPHeaderConfig(t *testing.T) {
	tests := []struct {
		name       string
		railway    string
		configured string
		override   string
		want       string
	}{
		{name: "default", want: "X-Client-IP"},
		{name: "railway", railway: "environment-id", want: "X-Real-IP"},
		{name: "explicit header", configured: "CF-Connecting-IP", want: "CF-Connecting-IP"},
		{name: "explicit railway header", railway: "environment-id", configured: "X-Client-IP", want: "X-Client-IP"},
		{name: "environment override", railway: "environment-id", configured: "X-Client-IP", override: "CF-Connecting-IP", want: "CF-Connecting-IP"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("RAILWAY_ENVIRONMENT_ID", tt.railway)
			t.Setenv("LIBREDESK_APP__SERVER__CLIENT_IP_HEADER", tt.override)
			config := koanf.New(".")
			if err := config.Set("app.server.client_ip_header", tt.configured); err != nil {
				t.Fatal(err)
			}
			if tt.override == "" {
				if err := os.Unsetenv("LIBREDESK_APP__SERVER__CLIENT_IP_HEADER"); err != nil {
					t.Fatal(err)
				}
			}
			initConfig(config)
			if got := config.String("app.server.client_ip_header"); got != tt.want {
				t.Fatalf("client_ip_header = %q, want %q", got, tt.want)
			}
		})
	}
}

func boolPointer(value bool) *bool {
	return &value
}
