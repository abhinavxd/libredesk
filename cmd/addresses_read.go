package main

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/jakedolan443/fernmail/internal/auth/models"
	"github.com/jakedolan443/fernmail/internal/envelope"
	wsmodels "github.com/jakedolan443/fernmail/internal/ws/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type addressReadResponse struct {
	AddressID int       `json:"address_id"`
	MarkedAt  time.Time `json:"marked_at"`
}

func handleMarkAddressRead(r *fastglue.Request) error {
	app := r.Context.(*App)
	auser := r.RequestCtx.UserValue("user").(models.User)
	addressID, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || addressID < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid address", nil, envelope.InputError)
	}
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	markedAt, err := app.conversation.MarkAddressRead(r.RequestCtx, user.ID, addressID, user.Permissions, user.Teams.IDs())
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	out := addressReadResponse{AddressID: addressID, MarkedAt: markedAt}
	if app.wsHub != nil {
		data, err := json.Marshal(wsmodels.Message{Type: "address_read", Data: out})
		if err == nil {
			app.wsHub.BroadcastMessage(wsmodels.BroadcastMessage{Users: []int{user.ID}, Data: data})
		}
	}
	return r.SendEnvelope(out)
}
