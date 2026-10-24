package models

import "testing"

func TestConfigUsesXOAUTH2(t *testing.T) {
	oauth := &OAuthConfig{Provider: "microsoft"}
	for _, tt := range []struct {
		name    string
		cfg     Config
		want    bool
		appOnly bool
	}{
		{"unset", Config{}, false, false},
		{"password", Config{AuthType: AuthTypePassword, OAuth: oauth}, false, false},
		{"delegated", Config{AuthType: AuthTypeOAuth2, OAuth: oauth}, true, false},
		{"delegated without oauth", Config{AuthType: AuthTypeOAuth2}, false, false},
		{"app-only", Config{AuthType: AuthTypeOAuth2App, OAuth: oauth}, true, true},
		{"app-only without oauth", Config{AuthType: AuthTypeOAuth2App}, false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.UsesXOAUTH2(); got != tt.want {
				t.Fatalf("UsesXOAUTH2() = %v, want %v", got, tt.want)
			}
			if got := tt.cfg.IsAppOnly(); got != tt.appOnly {
				t.Fatalf("IsAppOnly() = %v, want %v", got, tt.appOnly)
			}
		})
	}
}
