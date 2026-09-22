package main

import (
	"testing"

	hcmodels "github.com/abhinavxd/libredesk/internal/helpcenter/models"
)

func TestResolveColorScheme(t *testing.T) {
	const (
		system = hcmodels.ColorSchemeSystem
		light  = hcmodels.ColorSchemeLight
		dark   = hcmodels.ColorSchemeDark
	)
	tests := []struct {
		name      string
		scheme    string
		requested string
		embed     bool
		want      colorSchemeResult
	}{
		{"light site", light, "", false, colorSchemeResult{}},
		{"dark site", dark, "", false, colorSchemeResult{dark: true}},
		{"system site", system, "", false, colorSchemeResult{showToggle: true, followSystem: true}},
		{"preview dark of a light site", light, dark, false, colorSchemeResult{dark: true}},
		{"preview light of a dark site", dark, light, false, colorSchemeResult{}},
		{"preview of a system site keeps the toggle", system, dark, false, colorSchemeResult{dark: true, showToggle: true}},
		{"embed in a light widget on a dark site", dark, "", true, colorSchemeResult{}},
		{"embed in a dark widget", light, dark, true, colorSchemeResult{dark: true}},
		{"embed on a system site", system, dark, true, colorSchemeResult{dark: true}},
		{"unknown requested value", dark, "purple", false, colorSchemeResult{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveColorScheme(tt.scheme, tt.requested, tt.embed); got != tt.want {
				t.Errorf("resolveColorScheme(%q, %q, %v) = %+v, want %+v", tt.scheme, tt.requested, tt.embed, got, tt.want)
			}
		})
	}
}
