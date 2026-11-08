package main

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/helpcenter"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func TestAgentAccessRefusedOnHelpCenterDomain(t *testing.T) {
	db := testutil.NewDB(t, "cmd_middlewares")
	db.MustExec(`INSERT INTO help_centers (name, slug, custom_domain) VALUES ('Docs', 'docs', 'https://help.example.com')`)
	app := newValidatorTestApp(t)
	hc, err := helpcenter.New(helpcenter.Opts{DB: db, Lo: app.lo, I18n: app.i18n})
	if err != nil {
		t.Fatal(err)
	}
	app.helpcenter = hc

	page := func(r *fastglue.Request) error {
		t.Errorf("agent page served on %s", r.RequestCtx.Host())
		return nil
	}
	for name, h := range map[string]fastglue.FastRequestHandler{"authPage": authPage(page), "notAuthPage": notAuthPage(page)} {
		r := request("https://help.example.com/reset-password", "" /** ifNoneMatch **/)
		r.Context = app
		if err := h(r); err != nil {
			t.Fatal(err)
		}
		if got := r.RequestCtx.Response.StatusCode(); got != fasthttp.StatusNotFound {
			t.Errorf("%s status = %d, want %d", name, got, fasthttp.StatusNotFound)
		}
	}

	r := request("https://help.example.com/api/v1/users/me", "" /** ifNoneMatch **/)
	r.Context = app
	if _, err := authenticateUser(r, app); err == nil {
		t.Error("authenticateUser accepted a session on the help center domain")
	}
}
