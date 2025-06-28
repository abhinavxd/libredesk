package main

import (
	"strconv"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// returns all articles from the database.
func handleGetArticles(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	articles, err := app.article.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(articles)
}

// returns a single article by ID.
func handleGetArticle(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	article, err := app.article.GetByID(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(article)
}

// returns all articles for a specific section.
func handleGetArticlesBySection(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	sectionID, err := strconv.Atoi(r.RequestCtx.UserValue("sectionId").(string))
	if err != nil || sectionID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`sectionId`"), nil, envelope.InputError)
	}

	articles, err := app.article.GetBySectionID(sectionID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(articles)
}

// creates a new article in the database.
func handleCreateArticle(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req struct {
			Title     string `json:"title"`
			Content   string `json:"content"`
			SectionID int    `json:"section_id"`
		}
	)
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), err.Error(), envelope.InputError)
	}

	if req.Title == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`title`"), nil, envelope.InputError)
	}

	if req.SectionID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`section_id`"), nil, envelope.InputError)
	}

	id, err := app.article.Create(req.Title, req.Content, req.SectionID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(map[string]int{"id": id})
}

// deletes an article from the database.
func handleDeleteArticle(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	if err = app.article.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// updates an existing article in the database.
func handleUpdateArticle(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req struct {
			Title     string `json:"title"`
			Content   string `json:"content"`
			SectionID int    `json:"section_id"`
		}
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), err.Error(), envelope.InputError)
	}

	if req.Title == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`title`"), nil, envelope.InputError)
	}

	if req.SectionID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`section_id`"), nil, envelope.InputError)
	}

	if err = app.article.Update(id, req.Title, req.Content, req.SectionID); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// publishes an article.
func handlePublishArticle(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	if err = app.article.Publish(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// unpublishes an article.
func handleUnpublishArticle(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	if err = app.article.Unpublish(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}
