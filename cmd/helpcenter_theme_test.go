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

func TestBuildDarkThemeCSSVars(t *testing.T) {
	tests := []struct {
		name   string
		header hcmodels.HeaderTheme
		footer hcmodels.FooterTheme
		want   string
	}{
		{"nothing set", hcmodels.HeaderTheme{}, hcmodels.FooterTheme{}, ""},
		{"header text on default header", hcmodels.HeaderTheme{TextColor: "#000"}, hcmodels.FooterTheme{}, "--hc-header-text:initial;"},
		{"header text on solid header", hcmodels.HeaderTheme{BackgroundType: "solid", BackgroundColor: "#fff", TextColor: "#000"}, hcmodels.FooterTheme{}, ""},
		{"header text on image header", hcmodels.HeaderTheme{BackgroundType: "image", BackgroundImage: "/a.png", TextColor: "#000"}, hcmodels.FooterTheme{}, ""},
		{"header text on half-set gradient", hcmodels.HeaderTheme{BackgroundType: "gradient", GradientFrom: "#fff", TextColor: "#000"}, hcmodels.FooterTheme{}, "--hc-header-text:initial;"},
		{"footer text on default footer", hcmodels.HeaderTheme{}, hcmodels.FooterTheme{TextColor: "#000"}, "--hc-footer-text:initial;"},
		{"footer text on custom footer", hcmodels.HeaderTheme{}, hcmodels.FooterTheme{BackgroundColor: "#fff", TextColor: "#000"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(buildDarkThemeCSSVars(hcmodels.Theme{Header: tt.header, Footer: tt.footer}))
			if got != tt.want {
				t.Errorf("buildDarkThemeCSSVars() = %q, want %q", got, tt.want)
			}
		})
	}
}
