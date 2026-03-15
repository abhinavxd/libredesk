package main

import (
	"strconv"

	"github.com/abhinavxd/libredesk/internal/envelope"
	tmodels "github.com/abhinavxd/libredesk/internal/tag/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetTags returns all tags from the database.
func handleGetTags(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	t, err := app.tag.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(t)
}

// handleCreateTag creates a new tag in the database.
func handleCreateTag(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		tag = tmodels.Tag{}
	)
	if err := r.Decode(&tag, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}

	if tag.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`name`"), nil, envelope.InputError)
	}

	createdTag, err := app.tag.Create(tag.Name)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(createdTag)
}

// handleDeleteTag deletes a tag from the database.
func handleDeleteTag(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	if err = app.tag.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// handleUpdateTag updates an existing tag in the database.
func handleUpdateTag(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		tag = tmodels.Tag{}
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	if err := r.Decode(&tag, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}

	if tag.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`name`"), nil, envelope.InputError)
	}

	updatedTag, err := app.tag.Update(id, tag.Name)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(updatedTag)
}
