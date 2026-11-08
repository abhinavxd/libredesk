package main

import (
	"testing"
	"time"

	auth_ "github.com/abhinavxd/libredesk/internal/auth"
	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/helpcenter"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/user"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func TestAgentAccessRefusedOnHelpCenterDomain(t *testing.T) {
	db := testutil.NewDB(t, "cmd_middlewares")
	db.MustExec(`INSERT INTO help_centers (name, slug, custom_domain) VALUES ('Docs', 'docs', 'https://help.example.com')`)
	var agentID int
	if err := db.Get(&agentID, `INSERT INTO users (type, email, first_name, last_name, enabled) VALUES ('agent', 'agent@example.com', 'Agent', '', true) RETURNING id`); err != nil {
		t.Fatal(err)
	}
	app := newValidatorTestApp(t)
	hc, err := helpcenter.New(helpcenter.Opts{DB: db, Lo: app.lo, I18n: app.i18n})
	if err != nil {
		t.Fatal(err)
	}
	app.helpcenter = hc
	app.user, err = user.New(app.i18n, user.Opts{DB: db, Lo: app.lo})
	if err != nil {
		t.Fatal(err)
	}
	mr := miniredis.RunT(t)
	rd := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rd.Close() })
	app.auth, err = auth_.New(auth_.Config{SessionLifetime: time.Hour}, app.i18n, rd, app.lo, nil /** dialControl **/)
	if err != nil {
		t.Fatal(err)
	}
	initializedRequest := func(uri string) *fastglue.Request {
		ctx := &fasthttp.RequestCtx{}
		ctx.Init(&fasthttp.Request{}, nil /** remoteAddr **/, nil /** logger **/)
		ctx.Request.SetRequestURI(uri)
		return &fastglue.Request{RequestCtx: ctx, Context: app}
	}
	login := initializedRequest(testAppBaseURL + "/")
	if err := app.auth.SaveSession(amodels.User{ID: agentID, Email: "agent@example.com"}, login); err != nil {
		t.Fatal(err)
	}
	var cookie fasthttp.Cookie
	cookie.SetKey("libredesk_session")
	if !login.RequestCtx.Response.Header.Cookie(&cookie) {
		t.Fatal("missing session cookie")
	}
	sessionRequest := func(uri string) *fastglue.Request {
		r := initializedRequest(uri)
		r.RequestCtx.Request.Header.SetCookie("libredesk_session", string(cookie.Value()))
		return r
	}
	checkAppSession := func(t *testing.T) {
		t.Helper()
		u, err := authenticateUser(sessionRequest(testAppBaseURL+"/api/v1/users/me"), app)
		if err != nil || u.ID != agentID {
			t.Fatalf("app host rejected valid session: user=%d err=%v", u.ID, err)
		}
	}
	checkAppSession(t)
	page := func(r *fastglue.Request) error {
		t.Errorf("agent page served on %s", r.RequestCtx.Host())
		return nil
	}
	checkBlocked := func(t *testing.T, origin string) {
		t.Helper()
		for name, h := range map[string]fastglue.FastRequestHandler{"authPage": authPage(page), "notAuthPage": notAuthPage(page)} {
			r := sessionRequest(origin + "/reset-password")
			if err := h(r); err != nil {
				t.Fatal(err)
			}
			if got := r.RequestCtx.Response.StatusCode(); got != fasthttp.StatusNotFound {
				t.Errorf("%s status = %d, want %d", name, got, fasthttp.StatusNotFound)
			}
		}
		if _, err := authenticateUser(sessionRequest(origin+"/api/v1/users/me"), app); err == nil {
			t.Error("help center host accepted a valid agent session")
		}
	}
	for _, active := range []bool{true, false} {
		db.MustExec(`UPDATE help_centers SET is_active = $1`, active)
		for _, origin := range []string{"https://help.example.com", "https://HELP.EXAMPLE.COM:8443"} {
			checkBlocked(t, origin)
		}
		if _, ok := helpCenterByHost(app, "help.example.com"); ok != active {
			t.Fatalf("custom domain routing active=%v, want %v", ok, active)
		}
		checkAppSession(t)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := authenticateUser(sessionRequest("https://help.example.com/api/v1/users/me"), app); err == nil {
		t.Error("host lookup failure accepted an agent session")
	}
	for _, h := range []fastglue.FastRequestHandler{authPage(page), notAuthPage(page)} {
		r := sessionRequest("https://help.example.com/reset-password")
		if err := h(r); err != nil {
			t.Fatal(err)
		}
		if got := r.RequestCtx.Response.StatusCode(); got < 400 {
			t.Errorf("host lookup failure status = %d, want error response", got)
		}
	}
	checkAppSession(t)
}
