package main

import (
	"bytes"
	"html/template"
	"strings"
	"testing"

	pfmodels "github.com/abhinavxd/libredesk/internal/portalform/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
)

func TestPortalTicketMessageNavigation(t *testing.T) {
	for _, tc := range []struct {
		name       string
		page       int
		totalPages int
		older      bool
		newer      bool
	}{
		{"single page", 1, 1, false, false},
		{"latest page", 1, 3, true, false},
		{"middle page", 2, 3, true, true},
		{"oldest page", 3, 3, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmpl := template.Must(template.New("test").Parse(`{{define "portal-header"}}{{end}}{{define "portal-footer"}}{{end}}{{define "portal-status-badge"}}{{end}}`))
			tmpl = template.Must(tmpl.ParseFiles("../static/public/web-templates/portal/ticket.html"))
			var out bytes.Buffer
			err := tmpl.ExecuteTemplate(&out, "portal-ticket", map[string]any{
				"L": testutil.NewI18n(t),
				"Data": map[string]any{
					"ReferenceNumber": 42,
					"Page":            tc.page, "TotalPages": tc.totalPages,
					"PrevPage": tc.page - 1, "NextPage": tc.page + 1,
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(out.String(), ">Older messages</a>") != tc.older {
				t.Fatal("incorrect older messages navigation")
			}
			if strings.Contains(out.String(), ">Newer messages</a>") != tc.newer {
				t.Fatal("incorrect newer messages navigation")
			}
		})
	}
}

func TestPortalNumericFieldAllowsDecimals(t *testing.T) {
	tmpl := template.Must(template.New("test").Funcs(template.FuncMap{"AssetVer": func() string { return "test" }}).Parse(`{{define "portal-header"}}{{end}}{{define "portal-footer"}}{{end}}`))
	tmpl = template.Must(tmpl.ParseFiles("../static/public/web-templates/portal/new-ticket.html"))
	var out bytes.Buffer
	err := tmpl.ExecuteTemplate(&out, "portal-new-ticket", map[string]any{
		"L": testutil.NewI18n(t),
		"Data": map[string]any{
			"FormFields": []portalFieldView{{Field: pfmodels.Field{Key: "amount", Type: "number", Label: "Amount", Required: true}, Value: "12.50"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `type="number" name="field_amount" step="any" required`) || !strings.Contains(out.String(), `value="12.50"`) {
		t.Fatal("numeric field must preserve and accept decimal answers")
	}
}
