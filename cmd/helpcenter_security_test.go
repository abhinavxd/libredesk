package main

import (
	hcmodels "github.com/abhinavxd/libredesk/internal/helpcenter/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"html/template"
	"testing"
)

func TestHelpCenterCustomCodeRequiresIsolatedHost(t *testing.T) {
	for _, tt := range []struct {
		name, root, host, domain string
		want                     bool
	}{
		{"separate host", "https://app.example.com", "help.example.com", "https://help.example.com", true},
		{"dashboard", "https://app.example.com", "app.example.com", "https://help.example.com", false},
		{"no custom domain", "https://app.example.com", "app.example.com", "", false},
		{"same hostname", "https://app.example.com", "app.example.com:8443", "https://app.example.com:8443", false},
		{"case insensitive", "https://app.example.com", "HELP.example.com", "https://help.example.com", true},
		{"unrelated host", "https://app.example.com", "other.example.com", "https://help.example.com", false},
		{"missing root", "", "help.example.com", "https://help.example.com", false},
		{"malformed domain", "https://app.example.com", "help.example.com", ":invalid", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := helpCenterCustomCodeAllowed(tt.root, tt.host, tt.domain); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHelpCenterTemplateOmitsCustomCodeOnDashboard(t *testing.T) {
	app := &App{}
	app.consts.Store(&constants{AppBaseURL: "https://app.example.com"})
	hc := hcmodels.HelpCenter{CustomDomain: "https://help.example.com", CustomCSS: "</style><script>alert(1)</script>", CustomJS: "alert(1)"}
	for _, host := range []string{"app.example.com", "help.example.com"} {
		r := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}}
		r.RequestCtx.Request.SetRequestURI("https://" + host + "/hc/support/en")
		data := helpCenterTemplateData(app, r, hc, "en")
		wantCSS, wantJS := hc.CustomCSS, hc.CustomJS
		if host == "app.example.com" {
			wantCSS, wantJS = "", ""
		}
		if string(data["CustomCSS"].(template.CSS)) != wantCSS || string(data["CustomJS"].(template.JS)) != wantJS {
			t.Fatalf("unexpected custom code on %s", host)
		}
	}
}
