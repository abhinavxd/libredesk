package main

import (
	"strconv"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/helpcenter"
	hcmodels "github.com/abhinavxd/libredesk/internal/helpcenter/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// Help Centers

// handleGetHelpCenters returns all help centers from the database.
func handleGetHelpCenters(r *fastglue.Request) error {
	app := r.Context.(*App)
	helpCenters, err := app.helpcenter.GetAllHelpCenters()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(helpCenters)
}

// handleGetHelpCenter returns a specific help center by ID.
func handleGetHelpCenter(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	helpCenter, err := app.helpcenter.GetHelpCenterByID(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(helpCenter)
}

// handleCreateHelpCenter creates a new help center.
func handleCreateHelpCenter(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = helpcenter.HelpCenterCreateRequest{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	if err := validateHelpCenter(r, &req); err != nil {
		return err
	}

	helpCenter, err := app.helpcenter.CreateHelpCenter(req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(helpCenter)
}

// handleUpdateHelpCenter updates an existing help center.
func handleUpdateHelpCenter(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		req   = helpcenter.HelpCenterUpdateRequest{}
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	if err := validateHelpCenter(r, &req); err != nil {
		return err
	}

	helpCenter, err := app.helpcenter.UpdateHelpCenter(id, req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(helpCenter)
}

// handleDeleteHelpCenter deletes a help center.
func handleDeleteHelpCenter(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	if err := app.helpcenter.DeleteHelpCenter(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// Collections

// handleGetCollections returns all collections for a help center.
func handleGetCollections(r *fastglue.Request) error {
	var (
		app             = r.Context.(*App)
		helpCenterID, _ = strconv.Atoi(r.RequestCtx.UserValue("hc_id").(string))
		err             error
	)

	if helpCenterID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`help_center_id`"), nil, envelope.InputError)
	}

	// Check for locale filter
	locale := string(r.RequestCtx.QueryArgs().Peek("locale"))

	var collections []hcmodels.Collection
	if locale != "" {
		collections, err = app.helpcenter.GetCollectionsByHelpCenterAndLocale(helpCenterID, locale)
	} else {
		collections, err = app.helpcenter.GetCollectionsByHelpCenter(helpCenterID)
	}

	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(collections)
}

// handleGetCollection returns a specific collection by ID.
func handleGetCollection(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	collection, err := app.helpcenter.GetCollectionByID(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(collection)
}

// handleCreateCollection creates a new collection.
func handleCreateCollection(r *fastglue.Request) error {
	var (
		app               = r.Context.(*App)
		req               = helpcenter.CollectionCreateRequest{}
		helpCenterID, err = strconv.Atoi(r.RequestCtx.UserValue("hc_id").(string))
	)

	if helpCenterID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`help_center_id`"), nil, envelope.InputError)
	}

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	if err := validateCollection(r, &req); err != nil {
		return err
	}

	// Generate slug.
	req.Slug = stringutil.GenerateSlug(req.Name, true)

	collection, err := app.helpcenter.CreateCollection(helpCenterID, req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(collection)
}

// handleUpdateCollection updates an existing collection.
func handleUpdateCollection(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		req   = helpcenter.CollectionUpdateRequest{}
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)

	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	if err := validateCollection(r, &req); err != nil {
		return err
	}

	// Generate slug
	req.Slug = stringutil.GenerateSlug(req.Name, true)

	collection, err := app.helpcenter.UpdateCollection(id, req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(collection)
}

// handleDeleteCollection deletes a collection.
func handleDeleteCollection(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	if err := app.helpcenter.DeleteCollection(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleToggleCollection toggles the published status of a collection.
func handleToggleCollection(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)

	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	collection, err := app.helpcenter.ToggleCollectionPublished(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(collection)
}


// Articles

// handleGetArticles returns all articles for a collection.
func handleGetArticles(r *fastglue.Request) error {
	var (
		app             = r.Context.(*App)
		collectionID, _ = strconv.Atoi(r.RequestCtx.UserValue("col_id").(string))
		err             error
	)

	if collectionID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`collection_id`"), nil, envelope.InputError)
	}

	// Check for locale filter
	locale := string(r.RequestCtx.QueryArgs().Peek("locale"))

	var articles []hcmodels.Article
	if locale != "" {
		articles, err = app.helpcenter.GetArticlesByCollectionAndLocale(collectionID, locale)
	} else {
		articles, err = app.helpcenter.GetArticlesByCollection(collectionID)
	}

	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(articles)
}

// handleGetArticle returns a specific article by ID.
func handleGetArticle(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	article, err := app.helpcenter.GetArticleByID(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(article)
}

// handleCreateArticle creates a new article.
func handleCreateArticle(r *fastglue.Request) error {
	var (
		app             = r.Context.(*App)
		req             = helpcenter.ArticleCreateRequest{}
		collectionID, _ = strconv.Atoi(r.RequestCtx.UserValue("col_id").(string))
	)

	if collectionID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`collection_id`"), nil, envelope.InputError)
	}

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	if err := validateArticle(r, &req); err != nil {
		return err
	}

	// Generate slug
	req.Slug = stringutil.GenerateSlug(req.Title, true)

	if req.Status == "" {
		req.Status = hcmodels.ArticleStatusDraft
	}
	article, err := app.helpcenter.CreateArticle(collectionID, req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(article)
}

// handleUpdateArticle updates an existing article.
func handleUpdateArticle(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		req   = helpcenter.ArticleUpdateRequest{}
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)

	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	if err := validateArticle(r, &req); err != nil {
		return err
	}

	// Generate slug
	req.Slug = stringutil.GenerateSlug(req.Title, true)

	if req.Status == "" {
		req.Status = hcmodels.ArticleStatusDraft
	}

	article, err := app.helpcenter.UpdateArticle(id, req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(article)
}

// handleUpdateArticleByID updates an existing article by its ID (allows collection changes).
func handleUpdateArticleByID(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		req   = helpcenter.ArticleUpdateRequest{}
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)

	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	if err := validateArticle(r, &req); err != nil {
		return err
	}

	// Generate slug
	req.Slug = stringutil.GenerateSlug(req.Title, true)

	if req.Status == "" {
		req.Status = hcmodels.ArticleStatusDraft
	}

	article, err := app.helpcenter.UpdateArticle(id, req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(article)
}

// handleDeleteArticle deletes an article.
func handleDeleteArticle(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	if err := app.helpcenter.DeleteArticle(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// handleUpdateArticleStatus updates the status of an article.
func handleUpdateArticleStatus(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		req   = helpcenter.UpdateStatusRequest{}
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)

	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	if req.Status == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`status`"), nil, envelope.InputError)
	}

	article, err := app.helpcenter.UpdateArticleStatus(id, req.Status)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(article)
}


// handleGetHelpCenterTree returns the complete tree structure for a help center.
func handleGetHelpCenterTree(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)

	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	// Get locale from query parameter (optional)
	locale := string(r.RequestCtx.QueryArgs().Peek("locale"))

	tree, err := app.helpcenter.GetHelpCenterTree(id, locale)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(tree)
}

func validateHelpCenter(r *fastglue.Request, req any) error {
	app := r.Context.(*App)
	switch v := req.(type) {
	case *helpcenter.HelpCenterCreateRequest:
		if v.Name == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`name`"), nil, envelope.InputError)
		}
		if v.Slug == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`slug`"), nil, envelope.InputError)
		}
		if v.PageTitle == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`page_title`"), nil, envelope.InputError)
		}
		if v.DefaultLocale == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`default_locale`"), nil, envelope.InputError)
		}
	case *helpcenter.HelpCenterUpdateRequest:
		if v.Name == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`name`"), nil, envelope.InputError)
		}
		if v.Slug == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`slug`"), nil, envelope.InputError)
		}
		if v.PageTitle == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`page_title`"), nil, envelope.InputError)
		}
		if v.DefaultLocale == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`default_locale`"), nil, envelope.InputError)
		}
	}
	return nil
}

func validateCollection(r *fastglue.Request, req any) error {
	app := r.Context.(*App)
	switch v := req.(type) {
	case *helpcenter.CollectionCreateRequest:
		if v.Name == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`name`"), nil, envelope.InputError)
		}
		if v.Locale == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`locale`"), nil, envelope.InputError)
		}
	case *helpcenter.CollectionUpdateRequest:
		if v.Name == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`name`"), nil, envelope.InputError)
		}
		if v.Locale == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`locale`"), nil, envelope.InputError)
		}
	}
	return nil
}

func validateArticle(r *fastglue.Request, req any) error {
	app := r.Context.(*App)
	switch v := req.(type) {
	case *helpcenter.ArticleCreateRequest:
		if v.Title == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`title`"), nil, envelope.InputError)
		}
		if v.Content == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`content`"), nil, envelope.InputError)
		}
		if v.Locale == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`locale`"), nil, envelope.InputError)
		}
	case *helpcenter.ArticleUpdateRequest:
		if v.Title == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`title`"), nil, envelope.InputError)
		}
		if v.Content == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`content`"), nil, envelope.InputError)
		}
		if v.Locale == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`locale`"), nil, envelope.InputError)
		}
	}
	return nil
}
