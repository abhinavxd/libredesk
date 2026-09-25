package auth

import (
	"errors"
	"testing"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
	"github.com/zerodha/simplesessions/v3"
)

type sessionUsers struct {
	versions map[int]int
	err      error
}

func (u *sessionUsers) GetSessionVersion(id int) (int, error) {
	return u.versions[id], u.err
}

func TestSessionsRejectRevokedAndLegacyCookies(t *testing.T) {
	rd := redis.NewClient(&redis.Options{Addr: miniredis.RunT(t).Addr()})
	t.Cleanup(func() { rd.Close() })
	users := &sessionUsers{versions: map[int]int{1: 1, 2: 1}}
	lo := logf.New(logf.Opts{})
	a, err := New(Config{RootURL: func() (string, error) { return "https://app.example.com", nil }}, testutil.NewI18n(t), rd, &lo, nil /** dialControl **/, users)
	if err != nil {
		t.Fatal(err)
	}

	login := func(id, version int) string {
		t.Helper()
		r := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}}
		r.RequestCtx.Request.SetRequestURI("https://app.example.com/")
		if err := a.SaveSession(amodels.User{ID: id, SessionVersion: version}, r); err != nil {
			t.Fatal(err)
		}
		var cookie fasthttp.Cookie
		cookie.SetKey("libredesk_session")
		if !r.RequestCtx.Response.Header.Cookie(&cookie) {
			t.Fatal("missing session cookie")
		}
		return string(cookie.Value())
	}
	validate := func(cookie string, wantValid bool) {
		t.Helper()
		r := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}}
		r.RequestCtx.Request.SetRequestURI("https://app.example.com/")
		r.RequestCtx.Request.Header.SetCookie("libredesk_session", cookie)
		user, err := a.ValidateSession(r)
		if wantValid {
			if err != nil || user.ID <= 0 {
				t.Fatalf("valid session rejected: %v", err)
			}
		} else if !errors.Is(err, simplesessions.ErrInvalidSession) {
			t.Fatalf("revoked session accepted: user=%d err=%v", user.ID, err)
		}
	}

	first, second, other := login(1, 1), login(1, 1), login(2, 1)
	validate(first, true)
	validate(second, true)
	users.versions[1]++
	validate(first, false)
	validate(second, false)
	validate(other, true)
	validate(login(1, 2), true)
	validate(login(1, 0), false)
	users.err = errors.New("database unavailable")
	validate(other, false)
}

func TestSessionCookiesStayOnDashboardHost(t *testing.T) {
	rd := redis.NewClient(&redis.Options{Addr: miniredis.RunT(t).Addr()})
	t.Cleanup(func() { rd.Close() })
	lo := logf.New(logf.Opts{})
	users := &sessionUsers{versions: map[int]int{1: 1}}
	root := "https://app.example.com"
	a, err := New(Config{RootURL: func() (string, error) { return root, nil }}, testutil.NewI18n(t), rd, &lo, nil /** dialControl **/, users)
	if err != nil {
		t.Fatal(err)
	}
	r := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}}
	r.RequestCtx.Request.SetRequestURI(root)
	if err := a.SaveSession(amodels.User{ID: 1, SessionVersion: 1}, r); err != nil {
		t.Fatal(err)
	}
	var cookie fasthttp.Cookie
	cookie.SetKey("libredesk_session")
	r.RequestCtx.Response.Header.Cookie(&cookie)
	for _, host := range []string{"help.example.com", "other.example.com", "app.example.com.evil.example"} {
		t.Run(host, func(t *testing.T) {
			req := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}}
			req.RequestCtx.Request.SetRequestURI("https://" + host)
			req.RequestCtx.Request.Header.SetCookie("libredesk_session", string(cookie.Value()))
			if _, err := a.ValidateSession(req); err == nil {
				t.Fatal("dashboard session accepted on another host")
			}
			if err := a.SaveSession(amodels.User{ID: 1, SessionVersion: 1}, req); err == nil {
				t.Fatal("session issued on another host")
			}
		})
	}
	root = "https://new.example.com"
	if _, err := a.ValidateSession(r); err == nil {
		t.Fatal("old dashboard host still accepted after root URL change")
	}
}
