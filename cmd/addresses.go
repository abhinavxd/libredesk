package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strconv"
	"strings"

	"github.com/jakedolan443/fernmail/internal/address"
	"github.com/jakedolan443/fernmail/internal/auth/models"
	"github.com/jakedolan443/fernmail/internal/conversation"
	"github.com/jakedolan443/fernmail/internal/dbutil"
	"github.com/jakedolan443/fernmail/internal/envelope"
	"github.com/jakedolan443/fernmail/internal/inbox"
	imodels "github.com/jakedolan443/fernmail/internal/inbox/models"
	wsmodels "github.com/jakedolan443/fernmail/internal/ws/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func requireAddressAccess(app *App, userID, addressID int) error {
	allowed, err := app.address.CanAccess(userID, addressID)
	if err != nil {
		return envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if !allowed {
		return envelope.NewError(envelope.PermissionError, app.i18n.T("status.deniedPermission"), nil)
	}
	return nil
}

// handleGetAddresses gives the application sidebar only the addresses an agent
// can actually read. It intentionally exposes no transport credentials.
func handleGetAddresses(r *fastglue.Request) error {
	app := r.Context.(*App)
	user := r.RequestCtx.UserValue("user").(models.User)
	addresses, err := app.address.GetAccessible(user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(addresses)
}

func handleGetAddressConversations(r *fastglue.Request) error {
	app := r.Context.(*App)
	auser := r.RequestCtx.UserValue("user").(models.User)
	addressID, _ := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if addressID < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid address", nil, envelope.InputError)
	}
	if err := requireAddressAccess(app, auser.ID, addressID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	lists := conversation.ListsForUserPermissions(user.Permissions)
	if len(lists) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("status.deniedPermission"), nil, envelope.PermissionError)
	}
	filters := []dbutil.FilterNode{}
	if raw := r.RequestCtx.QueryArgs().Peek("filters"); len(raw) > 0 {
		if err := json.Unmarshal(raw, &filters); err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid address filters", nil, envelope.InputError)
		}
	}
	// This predicate is server-owned: callers cannot broaden an address list by
	// supplying their own filter tree.
	filters = append(filters, dbutil.FilterNode{
		Model: "conversations", Field: "address_id", Operator: "equals", Value: strconv.Itoa(addressID),
	})
	filterJSON, _ := json.Marshal(filters)
	page, size := getPagination(r)
	rows, err := app.conversation.GetConversations(user.ID, user.ID, user.Teams.IDs(), lists,
		string(r.RequestCtx.QueryArgs().Peek("order")), string(r.RequestCtx.QueryArgs().Peek("order_by")), string(filterJSON), page, size)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	total := 0
	if len(rows) > 0 {
		total = rows[0].Total
	}
	return r.SendEnvelope(envelope.PageResults{Results: rows, Total: total, PerPage: size, TotalPages: (total + size - 1) / size, Page: page})
}

func handleGetAdminAddresses(r *fastglue.Request) error {
	app := r.Context.(*App)
	addresses, err := app.address.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(addresses)
}

func handleGetAddressPrincipals(r *fastglue.Request) error {
	app := r.Context.(*App)
	principals, err := app.address.GetPrincipals()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(principals)
}

func handleGetAdminAddress(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, _ := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	entry, err := app.address.Get(id)
	if errors.Is(err, address.ErrNotFound) {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Address not found", nil, envelope.NotFoundError)
	}
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	access, err := app.address.GetAccess(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	entry.UserIDs, entry.TeamIDs = access.UserIDs, access.TeamIDs
	return r.SendEnvelope(entry)
}

func handleGetAddressAccess(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, _ := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	access, err := app.address.GetAccess(id)
	if errors.Is(err, address.ErrNotFound) {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Address not found", nil, envelope.NotFoundError)
	}
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(access)
}

func handleCreateAddress(r *fastglue.Request) error {
	app := r.Context.(*App)
	var input address.Address
	if err := r.Decode(&input, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid address", nil, envelope.InputError)
	}
	created, err := app.address.Create(input)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, envelope.InputError)
	}
	if err := syncEmailAddressConfig(app, created.InboxID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	broadcastAddressesUpdated(app)
	return r.SendEnvelope(created)
}

func handleUpdateAddress(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, _ := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if id < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid address", nil, envelope.InputError)
	}
	previous, err := app.address.Get(id)
	if errors.Is(err, address.ErrNotFound) {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Address not found", nil, envelope.NotFoundError)
	}
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	var input address.Address
	if err := r.Decode(&input, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid address", nil, envelope.InputError)
	}
	updated, err := app.address.Update(id, input)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, envelope.InputError)
	}
	if err := syncEmailAddressConfig(app, previous.InboxID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	if updated.InboxID != previous.InboxID {
		if err := syncEmailAddressConfig(app, updated.InboxID); err != nil {
			return sendErrorEnvelope(r, err)
		}
	}
	broadcastAddressesUpdated(app)
	return r.SendEnvelope(updated)
}

func handleDeleteAddress(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, _ := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	deleted, err := app.address.Delete(id)
	if errors.Is(err, address.ErrNotFound) {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Address not found", nil, envelope.NotFoundError)
	}
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, envelope.InputError)
	}
	if err := syncEmailAddressConfig(app, deleted.InboxID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	broadcastAddressesUpdated(app)
	return r.SendEnvelope(true)
}

// syncEmailAddressConfig is the one-way compatibility bridge to the IMAP
// receiver. email_addresses is canonical; runtime email_aliases is regenerated
// after each address mutation and the receiver is safely reloaded.
func syncEmailAddressConfig(app *App, inboxID int) error {
	record, err := app.inbox.GetDBRecord(inboxID)
	if err != nil {
		return err
	}
	if record.Channel != inbox.ChannelEmail {
		return fmt.Errorf("address transport is not an email inbox")
	}
	addresses, err := app.address.GetForInbox(inboxID)
	if err != nil {
		return err
	}
	if len(addresses) == 0 {
		return fmt.Errorf("email transport needs a mailbox address")
	}
	var config imodels.Config
	if err := json.Unmarshal(record.Config, &config); err != nil {
		return err
	}
	config.EmailAliases = runtimeEmailAliases(addresses)
	raw, err := json.Marshal(config)
	if err != nil {
		return err
	}
	if err := app.inbox.UpdateConfig(inboxID, raw); err != nil {
		return err
	}
	return reloadInbox(app, inboxID)
}

// provisionEmailAddresses promotes the primary transport address and any
// legacy configuration aliases into the canonical address model. It is used
// only when an email transport is first created; after that, the Addresses
// settings screen is the sole source of truth.
func provisionEmailAddresses(app *App, record imodels.Inbox, configured []imodels.EmailAlias) error {
	if record.Channel != inbox.ChannelEmail {
		return nil
	}
	primary, err := app.address.EnsureMailboxAddress(record.ID, record.From)
	if err != nil {
		return err
	}
	entries, err := app.address.GetForInbox(record.ID)
	if err != nil {
		return err
	}
	existing := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		existing[strings.ToLower(entry.Address)] = struct{}{}
	}
	for _, alias := range configured {
		candidate := strings.TrimSpace(alias.Address)
		if candidate == "" || strings.EqualFold(candidate, primary.Address) {
			continue
		}
		// Address.Create performs the authoritative syntactic validation and
		// detects an alias already claimed by a different transport.
		if _, found := existing[strings.ToLower(candidate)]; found {
			continue
		}
		displayName := strings.TrimSpace(alias.DisplayName)
		if displayName == "" {
			displayName = strings.TrimSpace(alias.Name)
		}
		created, err := app.address.Create(address.Address{
			InboxID: record.ID, Address: candidate, DisplayName: displayName,
			Kind: address.KindAlias, Enabled: alias.Enabled,
		})
		if err != nil {
			return err
		}
		existing[strings.ToLower(created.Address)] = struct{}{}
	}
	return syncEmailAddressConfig(app, record.ID)
}

// preserveAddressRuntimeConfig prevents a transport credential edit from
// covertly changing the user-facing endpoints. The current primary endpoint
// also makes the transport From identity immutable; moving it would make old
// conversations reply from the wrong address.
func preserveAddressRuntimeConfig(app *App, current imodels.Inbox, updated *imodels.Inbox) error {
	if current.Channel != inbox.ChannelEmail || updated.Channel != inbox.ChannelEmail {
		return nil
	}
	entries, err := app.address.GetForInbox(current.ID)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		if _, err := app.address.EnsureMailboxAddress(current.ID, current.From); err != nil {
			return err
		}
		entries, err = app.address.GetForInbox(current.ID)
		if err != nil {
			return err
		}
	}
	var primary *address.Address
	for i := range entries {
		if entries[i].Kind == address.KindMailbox {
			primary = &entries[i]
			break
		}
	}
	if primary == nil {
		return fmt.Errorf("email transport has no primary address")
	}
	parsed, err := mail.ParseAddress(updated.From)
	if err != nil || !strings.EqualFold(strings.TrimSpace(parsed.Address), primary.Address) {
		return fmt.Errorf("the transport From address is owned by its primary address; create a new transport instead")
	}
	var config imodels.Config
	if err := json.Unmarshal(updated.Config, &config); err != nil {
		return err
	}
	config.EmailAliases = runtimeEmailAliases(entries)
	raw, err := json.Marshal(config)
	if err != nil {
		return err
	}
	updated.Config = raw
	return nil
}

func runtimeEmailAliases(entries []address.Address) []imodels.EmailAlias {
	aliases := make([]imodels.EmailAlias, 0, len(entries))
	for _, entry := range entries {
		aliases = append(aliases, imodels.EmailAlias{
			Address: entry.Address, Name: entry.DisplayName, DisplayName: entry.DisplayName,
			Enabled: entry.Enabled, Default: entry.Kind == address.KindMailbox,
		})
	}
	return aliases
}

func broadcastAddressesUpdated(app *App) {
	app.wsHub.BroadcastMessage(wsmodels.BroadcastMessage{Data: []byte(`{"type":"addresses_updated"}`)})
}
