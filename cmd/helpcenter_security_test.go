package main

import (
	"html/template"
	"testing"

	hcmodels "github.com/abhinavxd/libredesk/internal/helpcenter/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func TestHelpCenterTemplateIncludesCustomCode(t *testing.T) {
	app := &App{}
	app.consts.Store(&constants{AppBaseURL: "https://app.example.com"})
	for _, tt := range []struct {
		name, host, domain string
	}{
		{"default domain", "app.example.com", ""},
		{"dashboard with custom domain", "app.example.com", "https://help.example.com"},
		{"custom domain", "help.example.com", "https://help.example.com"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			hc := hcmodels.HelpCenter{CustomDomain: tt.domain, CustomCSS: "body { color: inherit; }", CustomJS: "window.customHelpCenter = true;"}
			r := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}}
			r.RequestCtx.Request.SetRequestURI("https://" + tt.host + "/hc/support/en")
			data := helpCenterTemplateData(app, r, hc, "en")
			if string(data["CustomCSS"].(template.CSS)) != hc.CustomCSS || string(data["CustomJS"].(template.JS)) != hc.CustomJS {
				t.Fatal("configured custom code was omitted")
			}
		})
	}
}
