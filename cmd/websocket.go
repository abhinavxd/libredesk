package main

import (
	"net/url"
	"strings"
	"sync"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/ws"
	wsmodels "github.com/abhinavxd/libredesk/internal/ws/models"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

var wsWriteBufferPool = &sync.Pool{}

// agentUpgrader: same-origin only, with loopback allowed for dev.
var agentUpgrader = websocket.FastHTTPUpgrader{
	ReadBufferSize:  8192,
	WriteBufferSize: 8192,
	WriteBufferPool: wsWriteBufferPool,
	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
		origin := string(ctx.Request.Header.Peek("Origin"))
		if origin == "" {
			return false
		}
		u, err := url.Parse(origin)
		if err != nil || u.Host == "" {
			return false
		}
		isLocalhost := u.Hostname() == "localhost"
		if u.Scheme != "https" && !isLocalhost {
			return false
		}
		if strings.EqualFold(u.Host, string(ctx.Request.Host())) {
			return true
		}
		return isLocalhost
	},
	Error: ErrHandler,
}

// widgetUpgrader: cross-origin by design.
var widgetUpgrader = websocket.FastHTTPUpgrader{
	ReadBufferSize:  8192,
	WriteBufferSize: 8192,
	WriteBufferPool: wsWriteBufferPool,
	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
		return true
	},
	Error: ErrHandler,
}

// ErrHandler writes the handshake failure response.
func ErrHandler(ctx *fasthttp.RequestCtx, status int, reason error) {
	ctx.Error(reason.Error(), status)
}

// handleWS handles the websocket connection.
func handleWS(r *fastglue.Request, hub *ws.Hub) error {
	var (
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		app   = r.Context.(*App)
	)
	err := agentUpgrader.Upgrade(r.RequestCtx, func(conn *websocket.Conn) {
		c := ws.Client{
			ID:   auser.ID,
			Hub:  hub,
			Conn: conn,
			Send: make(chan wsmodels.WSMessage, 128),
		}
		hub.AddClient(&c)
		go c.Listen()
		c.Serve()
	})
	if err != nil {
		app.lo.Error("error upgrading tcp connection", "user_id", auser.ID, "error", err)
	}
	return nil
}
