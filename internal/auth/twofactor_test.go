package auth

import (
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

func TestPendingLoginIsOneUseAndExpires(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()
	logger := logf.New(logf.Opts{})
	a := &Auth{rd: client, i18n: testutil.NewI18n(t), logger: &logger}
	var ctx fasthttp.RequestCtx
	r := &fastglue.Request{RequestCtx: &ctx}
	expected := PendingLogin{UserID: 42, CredentialHash: "credential", Next: "/inboxes/assigned"}
	if err := a.BeginPendingLogin(r, expected); err != nil {
		t.Fatal(err)
	}
	var cookie fasthttp.Cookie
	cookie.SetKey(pendingLoginCookie)
	ctx.Response.Header.Cookie(&cookie)
	ctx.Request.Header.SetCookie(pendingLoginCookie, string(cookie.Value()))
	got, err := a.PendingLogin(r)
	if err != nil || got != expected {
		t.Fatalf("pending login mismatch: %+v %v", got, err)
	}
	if err := a.ConsumePendingLogin(r); err != nil {
		t.Fatal(err)
	}
	if _, err := a.PendingLogin(r); err == nil {
		t.Fatal("reused pending login")
	}
	if err := a.BeginPendingLogin(r, expected); err != nil {
		t.Fatal(err)
	}
	ctx.Response.Header.Cookie(&cookie)
	ctx.Request.Header.SetCookie(pendingLoginCookie, string(cookie.Value()))
	server.FastForward(6 * time.Minute)
	if _, err := a.PendingLogin(r); err == nil {
		t.Fatal("accepted expired pending login")
	}
}
