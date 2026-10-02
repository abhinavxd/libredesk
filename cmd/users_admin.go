package main

import (
	"strconv"

	amodels "github.com/jakedolan443/fernmail/internal/auth/models"
	"github.com/jakedolan443/fernmail/internal/envelope"
	"github.com/jakedolan443/fernmail/internal/user"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// The Users settings screen: one place to see everyone, set their role and
// addresses, and add, disable or delete people.

type managedUserCreateReq struct {
	FirstName        string `json:"first_name"`
	LastName         string `json:"last_name"`
	Email            string `json:"email"`
	Role             string `json:"role"`
	AddressIDs       []int  `json:"address_ids"`
	SendWelcomeEmail bool   `json:"send_welcome_email"`
}

type managedAddress struct {
	ID          int    `json:"id"`
	Address     string `json:"address"`
	DisplayName string `json:"display_name"`
	Kind        string `json:"kind"`
	Enabled     bool   `json:"enabled"`
}

type managedUsersResp struct {
	Users     []user.ManagedUser `json:"users"`
	Roles     []string           `json:"roles"`
	Addresses []managedAddress   `json:"addresses"`
	SelfID    int                `json:"self_id"`
}

func handleGetManagedUsers(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	users, err := app.user.GetManagedUsers()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	roles, err := app.user.GetRoleNames()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	entries, err := app.address.GetAllCompact()
	if err != nil {
		app.lo.Error("error fetching addresses for users screen", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	addresses := make([]managedAddress, 0, len(entries))
	for _, entry := range entries {
		addresses = append(addresses, managedAddress{ID: entry.ID, Address: entry.Address, DisplayName: entry.DisplayName, Kind: entry.Kind, Enabled: entry.Enabled})
	}
	return r.SendEnvelope(managedUsersResp{Users: users, Roles: roles, Addresses: addresses, SelfID: auser.ID})
}

func handleCreateManagedUser(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req managedUserCreateReq
	)
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	created, err := app.user.CreateManagedUser(req.FirstName, req.LastName, req.Email, req.Role, req.AddressIDs)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if req.SendWelcomeEmail {
		if err := sendWelcomeEmail(app, created.ID, created.Email); err != nil {
			return sendErrorEnvelope(r, err)
		}
	}
	return r.SendEnvelope(created)
}

func handleUpdateManagedUserAccess(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		req   user.AccessUpdate
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	if err := app.user.UpdateAccess(auser.ID, id, req); err != nil {
		return sendErrorEnvelope(r, err)
	}
	// Reconnect the user so live updates follow their new permissions.
	app.wsHub.KickUser(id)
	broadcastAddressesUpdated(app)
	updated, err := app.user.GetManagedUser(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(updated)
}

func handleDeleteManagedUser(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}
	if err := app.user.DeleteManagedUser(auser.ID, id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	app.wsHub.KickUser(id)
	return r.SendEnvelope(true)
}
