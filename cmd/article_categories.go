package main

import (
	"strconv"

	"github.com/abhinavxd/libredesk/internal/envelope"
	cmodels "github.com/abhinavxd/libredesk/internal/article_category/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// returns all article_categories from the database.
func handleGetArticleCategories(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	t, err := app.article_category.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(t)
}

// creates a new article_categories in the database.
func handleCreateArticleCategory(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		category = cmodels.Category{}
	)
	if err := r.Decode(&category, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), err.Error(), envelope.InputError)
	}

	if category.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`name`"), nil, envelope.InputError)
	}

	if err := app.article_category.Create(category.Name); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// deletes a article_categories from the database.
func handleDeleteArticleCategory(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	if err = app.article_category.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// updates an existing article_categories in the database.
func handleUpdateArticleCategory(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		category = cmodels.Category{}
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	if err := r.Decode(&category, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), err.Error(), envelope.InputError)
	}

	if category.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`name`"), nil, envelope.InputError)
	}

	if err = app.article_category.Update(id, category.Name); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}
