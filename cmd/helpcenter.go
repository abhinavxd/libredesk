package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"strconv"
	"strings"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/helpcenter"
	hcmodels "github.com/abhinavxd/libredesk/internal/helpcenter/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

const (
	publicSearchLimit     = 20
	popularArticlesLimit  = 6
	relatedArticlesLimit  = 5
	sitemapArticlesLimit  = 5000
	insightsTermLimit     = 20
	markdownSlugExtension = ".md"
	metaDescriptionLimit  = 160
)

// handleGetHelpCenters returns all help centers.
func handleGetHelpCenters(r *fastglue.Request) error {
	app := r.Context.(*App)
	helpCenters, err := app.helpcenter.GetAllHelpCenters()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(helpCenters)
}

// handleGetHelpCenter returns a help center by ID.
func handleGetHelpCenter(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`id`"), nil, envelope.InputError)
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
		req = helpcenter.HelpCenterRequest{}
	)
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
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

// handleUpdateHelpCenter updates a help center.
func handleUpdateHelpCenter(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		req   = helpcenter.HelpCenterRequest{}
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`id`"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
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

// handleToggleHelpCenterActive toggles whether a help center is live or paused.
func handleToggleHelpCenterActive(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`id`"), nil, envelope.InputError)
	}
	hc, err := app.helpcenter.ToggleHelpCenterActive(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(hc)
}

// handleDeleteHelpCenter deletes a help center.
func handleDeleteHelpCenter(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`id`"), nil, envelope.InputError)
	}
	if err := app.helpcenter.DeleteHelpCenter(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleGetHelpCenterTree returns the full collection/article tree for a help center.
func handleGetHelpCenterTree(r *fastglue.Request) error {
	var (
		app    = r.Context.(*App)
		id, _  = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		locale = strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("locale")))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`id`"), nil, envelope.InputError)
	}
	tree, err := app.helpcenter.GetHelpCenterTree(id, locale)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(tree)
}

// handleGetCollections returns all collections for a help center.
func handleGetCollections(r *fastglue.Request) error {
	var (
		app             = r.Context.(*App)
		helpCenterID, _ = strconv.Atoi(r.RequestCtx.UserValue("hc_id").(string))
	)
	if helpCenterID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`help_center_id`"), nil, envelope.InputError)
	}
	collections, err := app.helpcenter.GetCollectionsByHelpCenter(helpCenterID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(collections)
}

// handleGetCollection returns a collection by ID.
func handleGetCollection(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`id`"), nil, envelope.InputError)
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
		app             = r.Context.(*App)
		req             = helpcenter.CollectionRequest{}
		helpCenterID, _ = strconv.Atoi(r.RequestCtx.UserValue("hc_id").(string))
	)
	if helpCenterID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`help_center_id`"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	if err := validateCollection(r, &req); err != nil {
		return err
	}
	req.Slug = stringutil.GenerateSlug(req.Name, false)
	collection, err := app.helpcenter.CreateCollection(helpCenterID, req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(collection)
}

// handleUpdateCollection updates a collection, keeping its existing slug.
func handleUpdateCollection(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		req   = helpcenter.CollectionRequest{}
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`id`"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	if err := validateCollection(r, &req); err != nil {
		return err
	}
	existing, err := app.helpcenter.GetCollectionByID(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	req.Slug = existing.Slug
	collection, err := app.helpcenter.UpdateCollection(id, req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(collection)
}

// handleToggleCollection toggles the published status of a collection.
func handleToggleCollection(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`id`"), nil, envelope.InputError)
	}
	collection, err := app.helpcenter.ToggleCollectionPublished(id)
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
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`id`"), nil, envelope.InputError)
	}
	if err := app.helpcenter.DeleteCollection(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleGetArticles returns all articles for a collection.
func handleGetArticles(r *fastglue.Request) error {
	var (
		app             = r.Context.(*App)
		collectionID, _ = strconv.Atoi(r.RequestCtx.UserValue("col_id").(string))
	)
	if collectionID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`collection_id`"), nil, envelope.InputError)
	}
	articles, err := app.helpcenter.GetArticlesByCollection(collectionID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(articles)
}

// handleGetArticle returns an article by ID.
func handleGetArticle(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`id`"), nil, envelope.InputError)
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
		req             = helpcenter.ArticleRequest{}
		collectionID, _ = strconv.Atoi(r.RequestCtx.UserValue("col_id").(string))
	)
	if collectionID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`collection_id`"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	if err := validateArticle(r, &req); err != nil {
		return err
	}
	req.Slug = stringutil.GenerateSlug(req.Title, false)
	req.CollectionID = nil
	if req.Status == "" {
		req.Status = hcmodels.ArticleStatusDraft
	}
	if auser, ok := r.RequestCtx.UserValue("user").(amodels.User); ok && auser.ID > 0 {
		authorID := int64(auser.ID)
		req.AuthorID = &authorID
	}
	article, err := app.helpcenter.CreateArticle(collectionID, req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	// Best-effort; unlinked media is swept by the janitor after 7 days, so a failure
	// here must not fail the already-saved article.
	app.media.LinkHelpArticleMedia(article.ID, article.Content)
	return r.SendEnvelope(article)
}

// handleUpdateArticle updates an article, keeping its existing slug and collection.
func handleUpdateArticle(r *fastglue.Request) error {
	return updateArticle(r, false)
}

// handleUpdateArticleByID updates an article and allows moving it to another collection.
func handleUpdateArticleByID(r *fastglue.Request) error {
	return updateArticle(r, true)
}

// handleDeleteArticle deletes an article.
func handleDeleteArticle(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`id`"), nil, envelope.InputError)
	}
	if err := app.helpcenter.DeleteArticle(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleUpdateArticleStatus updates the status of an article.
func handleUpdateArticleStatus(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = struct {
			Status string `json:"status"`
		}{}
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`id`"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
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

// handleRedirectHelpCenterHome redirects bare /hc/{slug} to the default-locale home so the locale is always in the path.
func handleRedirectHelpCenterHome(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		slug = r.RequestCtx.UserValue("slug").(string)
	)
	helpCenter, err := app.helpcenter.GetHelpCenterBySlug(slug)
	if err != nil {
		return renderHelpCenterNotFound(r, slug)
	}
	r.RequestCtx.Redirect(fmt.Sprintf("/hc/%s/%s", slug, helpCenter.DefaultLocale), fasthttp.StatusFound)
	return nil
}

// handleShowHelpCenterHome renders the public help center home page.
func handleShowHelpCenterHome(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		slug = r.RequestCtx.UserValue("slug").(string)
	)
	helpCenter, err := app.helpcenter.GetHelpCenterBySlug(slug)
	if err != nil {
		return renderHelpCenterNotFound(r, slug)
	}
	locale := resolveLocale(r, helpCenter)
	tree, err := app.helpcenter.GetPublicTree(slug, locale)
	if err != nil {
		return renderHelpCenterNotFound(r, slug)
	}
	popular, err := app.helpcenter.GetPopularArticles(slug, locale, popularArticlesLimit)
	if err != nil {
		popular = nil
	}
	app.helpcenter.IncrementHelpCenterViewCount(tree.HelpCenter.ID)
	data := helpCenterTemplateData(tree.HelpCenter)
	addLocaleData(data, app, helpCenter, locale)
	return app.tmpl.RenderWebPage(r.RequestCtx, "help-center", map[string]interface{}{
		"Data": map[string]interface{}{
			"Title":           tree.HelpCenter.PageTitle,
			"MetaDescription": tree.HelpCenter.HeaderText,
			"CanonicalPath":   fmt.Sprintf("/hc/%s/%s", tree.HelpCenter.Slug, locale),
			"HelpCenter":      data,
			"Tree":            tree.Tree,
			"Popular":         popular,
		},
	})
}

// handleShowHelpCenterCollection renders a single collection's page: its sub-collections and articles.
func handleShowHelpCenterCollection(r *fastglue.Request) error {
	var (
		app            = r.Context.(*App)
		slug           = r.RequestCtx.UserValue("slug").(string)
		collectionSlug = r.RequestCtx.UserValue("collection_slug").(string)
	)
	helpCenter, err := app.helpcenter.GetHelpCenterBySlug(slug)
	if err != nil {
		return renderHelpCenterNotFound(r, slug)
	}
	locale := resolveLocale(r, helpCenter)
	tree, err := app.helpcenter.GetPublicTree(slug, locale)
	if err != nil {
		return renderHelpCenterNotFound(r, slug)
	}
	collection := findCollectionNode(tree.Tree, collectionSlug)
	if collection == nil {
		return renderHelpCenterNotFound(r, slug)
	}
	data := helpCenterTemplateData(helpCenter)
	addLocaleData(data, app, helpCenter, locale)
	return app.tmpl.RenderWebPage(r.RequestCtx, "help-collection", map[string]interface{}{
		"Data": map[string]interface{}{
			"Title":           fmt.Sprintf("%s - %s", collection.Name, helpCenter.Name),
			"MetaDescription": collection.Description,
			"CanonicalPath":   fmt.Sprintf("/hc/%s/%s/collections/%s", helpCenter.Slug, locale, collection.Slug),
			"CompactHero":     true,
			"HelpCenter":      data,
			"Collection":      collection,
		},
	})
}

// handleShowHelpCenterArticle renders a published article, or raw text for a `.md` slug.
func handleShowHelpCenterArticle(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		slug        = r.RequestCtx.UserValue("slug").(string)
		articleSlug = r.RequestCtx.UserValue("article_slug").(string)
		markdown    = strings.HasSuffix(articleSlug, markdownSlugExtension)
	)
	if markdown {
		articleSlug = strings.TrimSuffix(articleSlug, markdownSlugExtension)
	}
	helpCenter, err := app.helpcenter.GetHelpCenterBySlug(slug)
	if err != nil {
		return renderHelpCenterNotFound(r, slug)
	}
	locale := resolveLocale(r, helpCenter)
	article, err := app.helpcenter.GetPublishedArticle(slug, articleSlug, locale)
	if err != nil {
		return renderHelpCenterNotFound(r, slug)
	}
	app.helpcenter.IncrementArticleViewCount(article.ID)

	if markdown {
		r.RequestCtx.SetContentType("text/markdown; charset=utf-8")
		fmt.Fprintf(r.RequestCtx, "# %s\n\n%s\n", article.Title, stringutil.HTML2Text(article.Content))
		return nil
	}
	collection, err := app.helpcenter.GetCollectionByID(article.CollectionID)
	if err != nil {
		collection = hcmodels.Collection{}
	}
	related, err := app.helpcenter.GetPublishedArticlesByCollection(article.CollectionID, article.ID, relatedArticlesLimit)
	if err != nil {
		related = nil
	}
	metaDescription := article.MetaDescription
	if metaDescription == "" {
		metaDescription = articleExcerpt(article.Content)
	}
	metaTitle := article.MetaTitle
	if metaTitle == "" {
		metaTitle = fmt.Sprintf("%s - %s", article.Title, helpCenter.Name)
	}
	data := helpCenterTemplateData(helpCenter)
	addLocaleData(data, app, helpCenter, locale)
	return app.tmpl.RenderWebPage(r.RequestCtx, "help-article", map[string]interface{}{
		"Data": map[string]interface{}{
			"Title":           metaTitle,
			"MetaDescription": metaDescription,
			"OGImage":         article.MetaImageURL,
			"CanonicalPath":   fmt.Sprintf("/hc/%s/%s/articles/%s", helpCenter.Slug, locale, article.Slug),
			"OGType":          "article",
			"CompactHero":     true,
			"HelpCenter":      data,
			"Article":         article,
			"Collection":      collection,
			"Related":         related,
			"Content":         template.HTML(article.Content),
		},
	})
}

// handleHelpCenterSearch renders the public article search results page.
func handleHelpCenterSearch(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		slug  = r.RequestCtx.UserValue("slug").(string)
		query = strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("q")))
	)
	helpCenter, err := app.helpcenter.GetHelpCenterBySlug(slug)
	if err != nil {
		return renderHelpCenterNotFound(r, slug)
	}
	locale := resolveLocale(r, helpCenter)
	var articles []hcmodels.Article
	if query != "" {
		articles, err = app.helpcenter.SearchPublishedArticles(slug, query, locale, publicSearchLimit)
		if err != nil {
			articles = nil
		}
		app.helpcenter.LogSearch(helpCenter.ID, query, len(articles))
	}
	data := helpCenterTemplateData(helpCenter)
	addLocaleData(data, app, helpCenter, locale)
	return app.tmpl.RenderWebPage(r.RequestCtx, "help-search", map[string]interface{}{
		"Data": map[string]interface{}{
			"Title":      fmt.Sprintf("%s - %s", app.i18n.T("globals.terms.search"), helpCenter.Name),
			"NoIndex":    true,
			"HelpCenter": data,
			"Query":      query,
			"Results":    articles,
		},
	})
}

// handleHelpCenterSitemap serves a sitemap of all published articles in a help center.
func handleHelpCenterSitemap(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		slug = r.RequestCtx.UserValue("slug").(string)
	)
	helpCenter, err := app.helpcenter.GetHelpCenterBySlug(slug)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, app.i18n.T("globals.messages.notFound"), nil, envelope.NotFoundError)
	}
	locale := resolveLocale(r, helpCenter)
	articles, err := app.helpcenter.GetPopularArticles(slug, locale, sitemapArticlesLimit)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	rootURL, _ := app.setting.GetAppRootURL()
	type sitemapURL struct {
		Loc     string `xml:"loc"`
		LastMod string `xml:"lastmod,omitempty"`
	}
	type urlset struct {
		XMLName xml.Name     `xml:"urlset"`
		Xmlns   string       `xml:"xmlns,attr"`
		URLs    []sitemapURL `xml:"url"`
	}
	set := urlset{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	set.URLs = append(set.URLs, sitemapURL{Loc: fmt.Sprintf("%s/hc/%s/%s", rootURL, helpCenter.Slug, locale)})
	for _, a := range articles {
		set.URLs = append(set.URLs, sitemapURL{
			Loc:     fmt.Sprintf("%s/hc/%s/%s/articles/%s", rootURL, helpCenter.Slug, locale, a.Slug),
			LastMod: a.UpdatedAt.Format("2006-01-02"),
		})
	}
	out, err := xml.Marshal(set)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	r.RequestCtx.SetContentType("application/xml; charset=utf-8")
	fmt.Fprint(r.RequestCtx, xml.Header)
	r.RequestCtx.Write(out)
	return nil
}

// handleGetPublicHelpCenterTree returns the published-only tree as JSON.
func handleGetPublicHelpCenterTree(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		slug = r.RequestCtx.UserValue("slug").(string)
	)
	helpCenter, err := app.helpcenter.GetHelpCenterBySlug(slug)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	tree, err := app.helpcenter.GetPublicTree(slug, resolveLocale(r, helpCenter))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(tree)
}

// handleGetPublicHelpCenterArticle returns a published article as JSON.
func handleGetPublicHelpCenterArticle(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		slug        = r.RequestCtx.UserValue("slug").(string)
		articleSlug = r.RequestCtx.UserValue("article_slug").(string)
	)
	helpCenter, err := app.helpcenter.GetHelpCenterBySlug(slug)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	article, err := app.helpcenter.GetPublishedArticle(slug, articleSlug, resolveLocale(r, helpCenter))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	app.helpcenter.IncrementArticleViewCount(article.ID)
	return r.SendEnvelope(article)
}

// handlePublicHelpCenterSearch returns published article search results as JSON.
func handlePublicHelpCenterSearch(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		slug  = r.RequestCtx.UserValue("slug").(string)
		query = strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("q")))
	)
	helpCenter, err := app.helpcenter.GetHelpCenterBySlug(slug)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if query == "" {
		return r.SendEnvelope([]hcmodels.Article{})
	}
	articles, err := app.helpcenter.SearchPublishedArticles(slug, query, resolveLocale(r, helpCenter), publicSearchLimit)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	app.helpcenter.LogSearch(helpCenter.ID, query, len(articles))
	return r.SendEnvelope(articles)
}

// handleHelpCenterArticleFeedback records a reader's helpful/not-helpful vote for a published article.
func handleHelpCenterArticleFeedback(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		slug        = r.RequestCtx.UserValue("slug").(string)
		articleSlug = r.RequestCtx.UserValue("article_slug").(string)
		req         = struct {
			Helpful bool `json:"helpful"`
		}{}
	)
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	article, err := app.helpcenter.GetPublishedArticle(slug, articleSlug, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := app.helpcenter.RecordArticleFeedback(article.ID, req.Helpful); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleGetHelpCenterInsights returns search analytics for a help center.
func handleGetHelpCenterInsights(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`id`"), nil, envelope.InputError)
	}
	insights, err := app.helpcenter.GetInsights(id, insightsTermLimit)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(insights)
}

// updateArticle performs the shared article update flow; allowMove permits changing the collection.
func updateArticle(r *fastglue.Request, allowMove bool) error {
	var (
		app   = r.Context.(*App)
		req   = helpcenter.ArticleRequest{}
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`id`"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	if err := validateArticle(r, &req); err != nil {
		return err
	}
	existing, err := app.helpcenter.GetArticleByID(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	req.Slug = existing.Slug
	if !allowMove {
		req.CollectionID = nil
	}
	if req.Status == "" {
		req.Status = hcmodels.ArticleStatusDraft
	}
	article, err := app.helpcenter.UpdateArticle(id, req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	app.media.LinkHelpArticleMedia(article.ID, article.Content)
	return r.SendEnvelope(article)
}

// findCollectionNode returns the collection with the given slug from the tree, searching descendants.
func findCollectionNode(cols []hcmodels.TreeCollection, slug string) *hcmodels.TreeCollection {
	for i := range cols {
		if cols[i].Slug == slug {
			return &cols[i]
		}
		if found := findCollectionNode(cols[i].Children, slug); found != nil {
			return found
		}
	}
	return nil
}

// resolveLocale returns the locale path segment, falling back to the help center's default.
func resolveLocale(r *fastglue.Request, hc hcmodels.HelpCenter) string {
	if v, ok := r.RequestCtx.UserValue("locale").(string); ok {
		if loc := strings.TrimSpace(v); loc != "" {
			return loc
		}
	}
	return hc.DefaultLocale
}

// addLocaleData adds the current locale and the help center's configured languages (for the switcher) to the template data.
func addLocaleData(data map[string]interface{}, app *App, hc hcmodels.HelpCenter, current string) {
	locales := []string{}
	if len(hc.AllowedLocales) > 0 {
		if err := json.Unmarshal(hc.AllowedLocales, &locales); err != nil {
			locales = nil
		}
	}
	data["CurrentLocale"] = current
	data["AvailableLocales"] = locales
}

// helpCenterTemplateData shapes a help center row for the public templates.
func helpCenterTemplateData(hc hcmodels.HelpCenter) map[string]interface{} {
	navLinks := []hcmodels.NavLink{}
	if len(hc.NavLinks) > 0 {
		if err := json.Unmarshal(hc.NavLinks, &navLinks); err != nil {
			navLinks = nil
		}
	}
	theme := hcmodels.Theme{}
	if len(hc.Theme) > 0 {
		if err := json.Unmarshal(hc.Theme, &theme); err != nil {
			theme = hcmodels.Theme{}
		}
	}
	return map[string]interface{}{
		"Slug":          hc.Slug,
		"Name":          hc.Name,
		"PageTitle":     hc.PageTitle,
		"HeaderText":    hc.HeaderText,
		"LogoURL":       hc.LogoURL,
		"Color":         hc.Color,
		"DefaultLocale": hc.DefaultLocale,
		"NavLinks":      navLinks,
		"Theme":         theme,
		"ThemeCSS":      buildThemeCSSVars(theme),
		"CustomCSS":     template.CSS(hc.CustomCSS),
		"CustomJS":      template.JS(hc.CustomJS),
	}
}

// buildThemeCSSVars emits the theme's CSS custom-property overrides. Colors are
// hex-validated on save before reaching this trusted-CSS output.
func buildThemeCSSVars(t hcmodels.Theme) template.CSS {
	var b strings.Builder
	if t.Header.BackgroundType == "gradient" && t.Header.GradientFrom != "" && t.Header.GradientTo != "" {
		fmt.Fprintf(&b, "--hc-header-bg:linear-gradient(180deg,%s,%s);", t.Header.GradientFrom, t.Header.GradientTo)
	} else if t.Header.BackgroundType == "solid" && t.Header.BackgroundColor != "" {
		fmt.Fprintf(&b, "--hc-header-bg:%s;", t.Header.BackgroundColor)
	}
	if t.Header.TextColor != "" {
		fmt.Fprintf(&b, "--hc-header-text:%s;", t.Header.TextColor)
	}
	if t.Footer.BackgroundColor != "" {
		fmt.Fprintf(&b, "--hc-footer-bg:%s;", t.Footer.BackgroundColor)
	}
	if t.Footer.TextColor != "" {
		fmt.Fprintf(&b, "--hc-footer-text:%s;", t.Footer.TextColor)
	}
	return template.CSS(b.String())
}

// articleExcerpt returns a plain-text excerpt for meta description tags.
func articleExcerpt(htmlContent string) string {
	text := stringutil.HTML2Text(htmlContent)
	text = strings.Join(strings.Fields(text), " ")
	runes := []rune(text)
	if len(runes) <= metaDescriptionLimit {
		return text
	}
	text = string(runes[:metaDescriptionLimit])
	if i := strings.LastIndex(text, " "); i > 0 {
		text = text[:i]
	}
	return text
}

// renderHelpCenterNotFound renders the help center's themed 404, falling back to the
// generic error page when the help center slug itself is unknown.
func renderHelpCenterNotFound(r *fastglue.Request, slug string) error {
	app := r.Context.(*App)
	if helpCenter, err := app.helpcenter.GetHelpCenterBySlug(slug); err == nil {
		data := helpCenterTemplateData(helpCenter)
		addLocaleData(data, app, helpCenter, resolveLocale(r, helpCenter))
		rerr := app.tmpl.RenderWebPage(r.RequestCtx, "help-notfound", map[string]interface{}{
			"Data": map[string]interface{}{
				"Title":       app.i18n.T("helpCenter.notFoundTitle"),
				"NoIndex":     true,
				"CompactHero": true,
				"HelpCenter":  data,
			},
		})
		r.RequestCtx.SetStatusCode(fasthttp.StatusNotFound)
		return rerr
	}
	err := app.tmpl.RenderWebPage(r.RequestCtx, "error", map[string]interface{}{
		"Data": map[string]interface{}{
			"Title":        app.i18n.T("globals.messages.pageNotFound"),
			"ErrorMessage": app.i18n.T("globals.messages.pageNotFound"),
		},
	})
	r.RequestCtx.SetStatusCode(fasthttp.StatusNotFound)
	return err
}

func validateHelpCenter(r *fastglue.Request, req *helpcenter.HelpCenterRequest) error {
	app := r.Context.(*App)
	if req.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`name`"), nil, envelope.InputError)
	}
	if req.Slug == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`slug`"), nil, envelope.InputError)
	}
	if req.PageTitle == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`page_title`"), nil, envelope.InputError)
	}
	return nil
}

func validateCollection(r *fastglue.Request, req *helpcenter.CollectionRequest) error {
	app := r.Context.(*App)
	if req.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`name`"), nil, envelope.InputError)
	}
	return nil
}

func validateArticle(r *fastglue.Request, req *helpcenter.ArticleRequest) error {
	app := r.Context.(*App)
	if req.Title == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`title`"), nil, envelope.InputError)
	}
	if req.Content == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`content`"), nil, envelope.InputError)
	}
	return nil
}
