package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"hash/fnv"
	"html/template"
	"io"
	"log"
	"math"
	"net"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/helpcenter"
	hcmodels "github.com/abhinavxd/libredesk/internal/helpcenter/models"
	"github.com/abhinavxd/libredesk/internal/media"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	realip "github.com/ferluci/fast-realip"
	"github.com/knadh/stuffbin"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastcache/v4"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

const (
	publicSearchLimit     = 20
	popularArticlesLimit  = 6
	relatedArticlesLimit  = 5
	insightsTermLimit     = 20
	markdownSlugExtension = ".md"
	previewPageArticle    = "article"

	// sitemapURLLimit is the per-sitemap URL cap set by the sitemap protocol.
	sitemapURLLimit = 50000

	sitemapNamespace = "http://www.sitemaps.org/schemas/sitemap/0.9"
	sitemapDate      = "2006-01-02"
	schemaOrgContext = "https://schema.org"

	helpCenterCacheControl = "public, no-cache"

	fastCachePrefix = "libredesk:cache:"

	helpCenterCacheTTL = 30 * time.Minute

	helpCenterCacheGroup = "helpcenter"

	helpCenterCacheNamespaceKey = "hc_cache_ns"
	helpCenterCacheNamespace    = "hc"

	helpCenterXMLCacheControl = "public, max-age=300, stale-while-revalidate=3600"

	// articleFeedbackDedupTTL is how long a reader IP's vote on an article suppresses further votes.
	articleFeedbackDedupTTL = 24 * time.Hour

	noIndexHeader = "noindex"

	// hcHostRewriteKey marks a request already rewritten to its /hc/{slug} form.
	hcHostRewriteKey = "hc_host_rewrite"

	lucideSpritePath = "/static/public/static/lucide-sprite.svg"

	headerTextDark  = "#16181d"
	headerTextLight = "#ffffff"
)

var (
	// Logger is set because fastcache writes to the field when it finds it nil, racing across requests.
	helpCenterCacheOpts = &fastcache.Options{
		NamespaceKey:       helpCenterCacheNamespaceKey,
		ETag:               true,
		IncludeQueryString: true,
		TTL:                helpCenterCacheTTL,
		Logger:             log.New(io.Discard, "", 0),
	}

	// crawlerUARe matches search and preview bots, whose hits must not count as reader views.
	crawlerUARe = regexp.MustCompile(`(?i)bot\b|bot/|crawler|spider|crawling|slurp|facebookexternalhit|preview|headlesschrome|lighthouse|feedfetcher|python-requests|curl/|wget/`)

	lucideSymbolRe = regexp.MustCompile(`(?s)<symbol id="([a-z0-9-]+)"[^>]*>(.*?)</symbol>`)

	// rtlLanguages covers the right-to-left base languages an operator can put in AllowedLocales.
	rtlLanguages = []string{"ar", "he", "fa", "ur", "ps", "sd", "yi", "dv", "ckb"}

	// crawlerDisallowedPaths are the non-public parts of the app served on the same host as
	// the help center. /uploads is deliberately absent: article images live there and have
	// to stay crawlable for image search and social previews.
	crawlerDisallowedPaths = []string{
		"/api/",
		"/assets/",
		"/login",
		"/logout",
		"/reset-password",
		"/set-password",
		"/csat/",
		"/hc/*/*/search",
	}
)

// helpCenterCacheLogWriter routes fastcache's log lines into the app logger.
type helpCenterCacheLogWriter struct {
	lo *logf.Logger
}

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type urlset struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapRef struct {
	Loc string `xml:"loc"`
}

type sitemapIndex struct {
	XMLName  xml.Name     `xml:"sitemapindex"`
	Xmlns    string       `xml:"xmlns,attr"`
	Sitemaps []sitemapRef `xml:"sitemap"`
}

// localeLink is one entry in the language switcher or in the hreflang set.
type localeLink struct {
	Locale string
	Path   string
}

type previewTOCItem struct {
	ID    string
	Title string
}

func (w helpCenterCacheLogWriter) Write(p []byte) (int, error) {
	w.lo.Error("help center page cache: " + strings.TrimSpace(string(p)))
	return len(p), nil
}

// handleGetHelpCenterLocales returns the locales a help center can be authored in.
func handleGetHelpCenterLocales(r *fastglue.Request) error {
	return r.SendEnvelope(helpcenter.SupportedLocales())
}

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
	if err := validateHelpCenter(app, &req); err != nil {
		return sendErrorEnvelope(r, err)
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
	if err := validateHelpCenter(app, &req); err != nil {
		return sendErrorEnvelope(r, err)
	}
	helpCenter, err := app.helpcenter.UpdateHelpCenter(id, req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(helpCenter)
}

// handleHelpCenterPreview renders the home page from unsaved settings so the admin can preview edits.
func handleHelpCenterPreview(r *fastglue.Request) error {
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
	helpCenter, err := app.helpcenter.DraftHelpCenter(id, req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	locale := helpCenter.DefaultLocale
	if string(r.RequestCtx.QueryArgs().Peek("page")) == previewPageArticle {
		return renderHelpCenterArticlePreview(r, helpCenter, locale)
	}
	popular, err := app.helpcenter.GetPopularArticles(helpCenter.Slug, locale, popularArticlesLimit)
	if err != nil {
		popular = nil
	}
	tree, err := app.helpcenter.GetPublicTree(helpCenter, locale)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := app.tmpl.RenderWebPage(r.RequestCtx, hcPageName(helpCenter, "help-center"), map[string]interface{}{
		"L": localeI18n(app, locale),
		"Data": map[string]interface{}{
			"Title":       helpCenter.PageTitle,
			"LandingHero": true,
			"HelpCenter":  helpCenterTemplateData(app, helpCenter, locale),
			"Tree":        tree.Tree,
			"Popular":     popular,
		},
	}); err != nil {
		return sendErrorEnvelope(r, err)
	}
	r.RequestCtx.Response.Header.Set("Cache-Control", "no-store")
	return nil
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
	if err := validateCollection(app, &req); err != nil {
		return sendErrorEnvelope(r, err)
	}
	req.Slug = helpcenter.GenerateSlug(req.Name)
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
	if err := validateCollection(app, &req); err != nil {
		return sendErrorEnvelope(r, err)
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

// handleUpdateCollectionSortOrders reorders the collections of a help center.
func handleUpdateCollectionSortOrders(r *fastglue.Request) error {
	var (
		app             = r.Context.(*App)
		helpCenterID, _ = strconv.Atoi(r.RequestCtx.UserValue("hc_id").(string))
		orders          = make(map[int]int)
	)
	if helpCenterID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`help_center_id`"), nil, envelope.InputError)
	}
	if err := r.Decode(&orders, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	if err := app.helpcenter.UpdateCollectionSortOrders(helpCenterID, orders); err != nil {
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
	if err := validateArticle(app, &req); err != nil {
		return sendErrorEnvelope(r, err)
	}
	req.Slug = helpcenter.GenerateSlug(req.Title)
	req.CollectionID = nil
	if req.Status == "" {
		req.Status = hcmodels.ArticleStatusDraft
	}
	if auser, ok := r.RequestCtx.UserValue("user").(amodels.User); ok && auser.ID > 0 {
		userID := int64(auser.ID)
		req.CreatedBy = &userID
		if req.AuthorID == nil {
			req.AuthorID = &userID
		}
	}
	article, err := app.helpcenter.CreateArticle(collectionID, req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := app.media.LinkHelpArticleMedia(article.ID, article.Content); err != nil {
		app.lo.Error("error linking help article media", "article_id", article.ID, "error", err)
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
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`id`"), nil, envelope.InputError)
	}
	if err := app.helpcenter.DeleteArticle(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleMoveArticle moves an article to another collection.
func handleMoveArticle(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = struct {
			CollectionID int `json:"collection_id"`
		}{}
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`id`"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	if req.CollectionID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`collection_id`"), nil, envelope.InputError)
	}
	article, err := app.helpcenter.MoveArticle(id, req.CollectionID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(article)
}

// handleUpdateArticleSortOrders reorders the articles of a collection.
func handleUpdateArticleSortOrders(r *fastglue.Request) error {
	var (
		app             = r.Context.(*App)
		collectionID, _ = strconv.Atoi(r.RequestCtx.UserValue("col_id").(string))
		orders          = make(map[int]int)
	)
	if collectionID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`collection_id`"), nil, envelope.InputError)
	}
	if err := r.Decode(&orders, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	if err := app.helpcenter.UpdateArticleSortOrders(collectionID, orders); err != nil {
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
		return renderHelpCenterPageError(r, nil, err)
	}
	if redirectHelpCenterCanonicalHost(r, helpCenter) {
		return nil
	}
	// 302, not 301: the default locale is mutable and browsers cache permanent redirects.
	redirectPath(r.RequestCtx, helpCenterHomePath(helpCenter, helpCenter.DefaultLocale), fasthttp.StatusFound)
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
		return renderHelpCenterPageError(r, nil, err)
	}
	if redirectHelpCenterCanonicalHost(r, helpCenter) {
		return nil
	}
	locale, ok := resolveLocale(r, helpCenter)
	if !ok {
		return renderHelpCenterNotFound(r, &helpCenter)
	}
	tree, err := app.helpcenter.GetPublicTree(helpCenter, locale)
	if err != nil {
		return renderHelpCenterPageError(r, &helpCenter, err)
	}
	popular, err := app.helpcenter.GetPopularArticles(slug, locale, popularArticlesLimit)
	if err != nil {
		popular = nil
	}
	var (
		root            = helpCenterBaseURL(app, helpCenter)
		locales         = helpCenterLocales(helpCenter)
		pathFor         = func(l string) string { return helpCenterHomePath(helpCenter, l) }
		theme           = helpCenterTheme(tree.HelpCenter)
		metaDescription = firstNonEmpty(tree.HelpCenter.MetaDescription, theme.Header.Heading)
	)
	data := helpCenterTemplateData(app, tree.HelpCenter, locale)
	return renderHelpCenterPage(r, hcPageName(helpCenter, "help-center"), map[string]interface{}{
		"L": localeI18n(app, locale),
		"Data": map[string]interface{}{
			"Title":           tree.HelpCenter.PageTitle,
			"MetaDescription": metaDescription,
			"CanonicalPath":   pathFor(locale),
			"LandingHero":     true,
			"OGImage":         absoluteURL(root, publicAssetPaths(app, theme.LogoURL)),
			"Alternates":      helpCenterAlternates(helpCenter, locales, pathFor),
			"XDefaultPath":    defaultLocalePath(helpCenter, locales, pathFor),
			"LocaleLinks":     helpCenterLocaleLinks(helpCenter, locales, pathFor),
			"JSONLD":          homeJSONLD(root, tree.HelpCenter, locale),
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
		return renderHelpCenterPageError(r, nil, err)
	}
	if redirectHelpCenterCanonicalHost(r, helpCenter) {
		return nil
	}
	locale, ok := resolveLocale(r, helpCenter)
	if !ok {
		return renderHelpCenterNotFound(r, &helpCenter)
	}
	tree, err := app.helpcenter.GetPublicTree(helpCenter, locale)
	if err != nil {
		return renderHelpCenterPageError(r, &helpCenter, err)
	}
	collection := findCollectionNode(tree.Tree, collectionSlug)
	if collection == nil {
		return renderHelpCenterNotFound(r, &helpCenter)
	}
	translated, err := app.helpcenter.GetPublishedCollectionLocales(slug, collection.Slug)
	if err != nil {
		translated = []string{locale}
	}
	var (
		root    = helpCenterBaseURL(app, helpCenter)
		pathFor = func(l string) string { return collectionPath(helpCenter, l, collection.Slug) }
	)
	data := helpCenterTemplateData(app, helpCenter, locale)
	return renderHelpCenterPage(r, hcPageName(helpCenter, "help-collection"), map[string]interface{}{
		"L": localeI18n(app, locale),
		"Data": map[string]interface{}{
			"Title":            fmt.Sprintf("%s - %s", collection.Name, helpCenter.Name),
			"MetaDescription":  collection.Description,
			"CanonicalPath":    pathFor(locale),
			"OGImage":          absoluteURL(root, publicAssetPaths(app, helpCenterTheme(helpCenter).LogoURL)),
			"Alternates":       helpCenterAlternates(helpCenter, translated, pathFor),
			"XDefaultPath":     defaultLocalePath(helpCenter, translated, pathFor),
			"LocaleLinks":      helpCenterLocaleLinks(helpCenter, translated, pathFor),
			"JSONLD":           collectionJSONLD(root, helpCenter, *collection, locale, pathFor(locale)),
			"HelpCenter":       data,
			"Collection":       collection,
			"Tree":             tree.Tree,
			"ActiveCollection": collection.Slug,
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
		return renderHelpCenterPageError(r, nil, err)
	}
	if redirectHelpCenterCanonicalHost(r, helpCenter) {
		return nil
	}
	locale, ok := resolveLocale(r, helpCenter)
	if !ok {
		return renderHelpCenterNotFound(r, &helpCenter)
	}
	article, err := app.helpcenter.GetPublishedArticle(slug, articleSlug, locale)
	if err != nil {
		return renderHelpCenterPageError(r, &helpCenter, err)
	}
	// The JSON-LD embeds the author too, not just the byline.
	hideArticleAuthor(helpCenterTheme(helpCenter), &article)

	if markdown {
		r.RequestCtx.SetContentType("text/markdown; charset=utf-8")
		fmt.Fprintf(r.RequestCtx, "# %s\n\n%s\n", article.Title, stringutil.HTML2Text(article.Content))
		return nil
	}
	collection, err := app.helpcenter.GetCollectionByID(article.CollectionID)
	if err != nil {
		collection = hcmodels.Collection{}
	}
	related, err := app.helpcenter.GetPublishedArticlesByCollection(article.CollectionID, article.ID, article.Locale, relatedArticlesLimit)
	if err != nil {
		related = nil
	}
	translated, err := app.helpcenter.GetPublishedArticleLocales(slug, article.Slug)
	if err != nil {
		translated = []string{locale}
	}
	var (
		root            = helpCenterBaseURL(app, helpCenter)
		pathFor         = func(l string) string { return articlePath(helpCenter, l, article.Slug) }
		metaDescription = firstNonEmpty(article.MetaDescription, article.Excerpt)
		metaTitle       = firstNonEmpty(article.MetaTitle, fmt.Sprintf("%s - %s", article.Title, helpCenter.Name))
		ogImage         = absoluteURL(root, publicAssetPaths(app, firstNonEmpty(article.MetaImageURL, helpCenterTheme(helpCenter).LogoURL)))
	)
	data := helpCenterTemplateData(app, helpCenter, locale)
	return renderHelpCenterPage(r, hcPageName(helpCenter, "help-article"), map[string]interface{}{
		"L": localeI18n(app, locale),
		"Data": map[string]interface{}{
			"Title":            metaTitle,
			"MetaDescription":  metaDescription,
			"OGImage":          ogImage,
			"CanonicalPath":    pathFor(locale),
			"OGType":           "article",
			"PublishedTime":    article.CreatedAt.Format(time.RFC3339),
			"ModifiedTime":     article.UpdatedAt.Format(time.RFC3339),
			"Alternates":       helpCenterAlternates(helpCenter, translated, pathFor),
			"XDefaultPath":     defaultLocalePath(helpCenter, translated, pathFor),
			"LocaleLinks":      helpCenterLocaleLinks(helpCenter, translated, pathFor),
			"JSONLD":           articleJSONLD(root, helpCenter, collection, article, locale, pathFor(locale), ogImage),
			"HelpCenter":       data,
			"Article":          article,
			"AuthorInitial":    authorInitial(article),
			"Collection":       collection,
			"Related":          related,
			"Tree":             sidebarTree(app, helpCenter, locale),
			"ActiveCollection": collection.Slug,
			"ActiveArticle":    article.Slug,
			"Content":          template.HTML(publicAssetPaths(app, stringutil.DeferOffscreenImages(article.Content))),
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
		return renderHelpCenterPageError(r, nil, err)
	}
	if redirectHelpCenterCanonicalHost(r, helpCenter) {
		return nil
	}
	locale, ok := resolveLocale(r, helpCenter)
	if !ok {
		return renderHelpCenterNotFound(r, &helpCenter)
	}
	var articles []hcmodels.Article
	if query != "" {
		articles, err = app.helpcenter.SearchPublishedArticles(slug, query, locale, publicSearchLimit)
		if err != nil {
			articles = nil
		} else if !isCrawler(r) {
			app.helpcenter.LogSearch(helpCenter.ID, query, len(articles))
		}
	}
	var (
		data    = helpCenterTemplateData(app, helpCenter, locale)
		lcl     = localeI18n(app, locale)
		pathFor = func(l string) string { return searchPath(helpCenter, l) }
	)
	return renderHelpCenterPage(r, hcPageName(helpCenter, "help-search"), map[string]interface{}{
		"L": lcl,
		"Data": map[string]interface{}{
			"Title":       fmt.Sprintf("%s - %s", lcl.T("globals.terms.search"), helpCenter.Name),
			"NoIndex":     true,
			"LocaleLinks": helpCenterLocaleLinks(helpCenter, helpCenterLocales(helpCenter), pathFor),
			"HelpCenter":  data,
			"Query":       query,
			"Results":     articles,
			"Tree":        sidebarTree(app, helpCenter, locale),
		},
	})
}

// handleHelpCenterSitemap serves a sitemap of a help center's published pages in one locale.
func handleHelpCenterSitemap(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		slug = r.RequestCtx.UserValue("slug").(string)
	)
	helpCenter, err := app.helpcenter.GetHelpCenterBySlug(slug)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, app.i18n.T("globals.messages.notFound"), nil, envelope.NotFoundError)
	}
	if redirectHelpCenterCanonicalHost(r, helpCenter) {
		return nil
	}
	locale, ok := resolveLocale(r, helpCenter)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, app.i18n.T("globals.messages.notFound"), nil, envelope.NotFoundError)
	}
	articles, err := app.helpcenter.GetPopularArticles(slug, locale, sitemapURLLimit)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if len(articles) == sitemapURLLimit {
		app.lo.Warn("help center sitemap hit the URL limit, some articles are missing", "help_center_slug", slug, "locale", locale, "limit", sitemapURLLimit)
	}
	tree, err := app.helpcenter.GetPublicTree(helpCenter, locale)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	root := helpCenterBaseURL(app, helpCenter)
	set := urlset{Xmlns: sitemapNamespace}
	set.URLs = append(set.URLs, sitemapURL{
		Loc:     root + helpCenterHomePath(helpCenter, locale),
		LastMod: helpCenter.UpdatedAt.Format(sitemapDate),
	})
	for _, c := range flattenCollections(tree.Tree, nil) {
		set.URLs = append(set.URLs, sitemapURL{
			Loc:     root + collectionPath(helpCenter, locale, c.Slug),
			LastMod: c.UpdatedAt.Format(sitemapDate),
		})
	}
	for _, a := range articles {
		set.URLs = append(set.URLs, sitemapURL{
			Loc:     root + articlePath(helpCenter, locale, a.Slug),
			LastMod: a.UpdatedAt.Format(sitemapDate),
		})
	}
	if len(set.URLs) > sitemapURLLimit {
		app.lo.Warn("help center sitemap truncated to the URL limit", "help_center_slug", slug, "locale", locale, "limit", sitemapURLLimit)
		set.URLs = set.URLs[:sitemapURLLimit]
	}
	return sendXML(r, set)
}

// handleSitemapIndex lists the per-locale sitemaps of the help centers served on this host.
func handleSitemapIndex(r *fastglue.Request) error {
	app := r.Context.(*App)
	helpCenters, err := app.helpcenter.GetActiveHelpCenters()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	index := sitemapIndex{Xmlns: sitemapNamespace}
	for _, hc := range helpCentersForHost(r, helpCenters) {
		root := helpCenterBaseURL(app, hc)
		for _, locale := range helpCenterLocales(hc) {
			index.Sitemaps = append(index.Sitemaps, sitemapRef{Loc: fmt.Sprintf("%s%s/sitemap.xml", root, helpCenterHomePath(hc, locale))})
		}
	}
	return sendXML(r, index)
}

// handleRobotsTxt keeps crawlers out of the agent app and points them at this host's sitemap index.
func handleRobotsTxt(r *fastglue.Request) error {
	app := r.Context.(*App)
	r.RequestCtx.SetContentType("text/plain; charset=utf-8")
	fmt.Fprint(r.RequestCtx, "User-agent: *\n")
	for _, path := range crawlerDisallowedPaths {
		fmt.Fprintf(r.RequestCtx, "Disallow: %s\n", path)
	}
	helpCenters, err := app.helpcenter.GetActiveHelpCenters()
	if err != nil {
		return nil
	}
	// Crawlers ignore a Sitemap directive on another host, and listing one publishes the other hosts.
	hosted := helpCentersForHost(r, helpCenters)
	if len(hosted) > 0 && hosted[0].CustomDomain != "" {
		fmt.Fprint(r.RequestCtx, "Disallow: /*/search\n")
	}
	if len(hosted) > 0 {
		fmt.Fprintf(r.RequestCtx, "\nSitemap: %s/sitemap.xml\n", helpCenterHostRootURL(app, hosted))
	}
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
	locale := resolveQueryLocale(r, helpCenter)
	tree, err := app.helpcenter.GetPublicTree(helpCenter, locale)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	hideTreeAuthors(helpCenterTheme(helpCenter), tree.Tree)
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
	locale := resolveQueryLocale(r, helpCenter)
	article, err := app.helpcenter.GetPublishedArticle(slug, articleSlug, locale)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	hideArticleAuthor(helpCenterTheme(helpCenter), &article)
	if !isCrawler(r) {
		app.helpcenter.IncrementArticleViewCount(article.ID)
	}
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
	locale := resolveQueryLocale(r, helpCenter)
	articles, err := app.helpcenter.SearchPublishedArticles(slug, query, locale, publicSearchLimit)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	// The typeahead passes log=0 per keystroke and logs the settled term with a later log=1 call.
	if string(r.RequestCtx.QueryArgs().Peek("log")) != "0" {
		app.helpcenter.LogSearch(helpCenter.ID, query, len(articles))
	}
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
	helpCenter, err := app.helpcenter.GetHelpCenterBySlug(slug)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	article, err := app.helpcenter.GetPublishedArticle(slug, articleSlug, resolveQueryLocale(r, helpCenter))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	dedupKey := fmt.Sprintf("help_center:feedback:%d:%s", article.ID, realip.FromRequest(r.RequestCtx))
	if ok, err := app.redis.SetNX(r.RequestCtx, dedupKey, "1", articleFeedbackDedupTTL).Result(); err == nil && !ok {
		return r.SendEnvelope(true)
	}
	if err := app.helpcenter.RecordArticleFeedback(article.ID, req.Helpful); err != nil {
		// A failed insert must not consume the reader's vote for the dedup window.
		app.redis.Del(r.RequestCtx, dedupKey)
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

// handleUpdateArticle updates an article, keeping its existing slug.
func handleUpdateArticle(r *fastglue.Request) error {
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
	if err := validateArticle(app, &req); err != nil {
		return sendErrorEnvelope(r, err)
	}
	existing, err := app.helpcenter.GetArticleByID(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	req.Slug = existing.Slug
	if req.CollectionID != nil && *req.CollectionID != existing.CollectionID {
		from, err := app.helpcenter.GetCollectionByID(existing.CollectionID)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		to, err := app.helpcenter.GetCollectionByID(*req.CollectionID)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		if from.HelpCenterID != to.HelpCenterID {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("helpCenter.invalidParent"), nil, envelope.InputError)
		}
	}
	if req.Status == "" {
		req.Status = hcmodels.ArticleStatusDraft
	}
	article, err := app.helpcenter.UpdateArticle(id, req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := app.media.LinkHelpArticleMedia(article.ID, article.Content); err != nil {
		app.lo.Error("error linking help article media", "article_id", article.ID, "error", err)
	}
	return r.SendEnvelope(article)
}

// renderHelpCenterPage renders a public help center page as cacheable, overriding the
// no-store default that RenderWebPage applies for the app's authenticated pages.
func renderHelpCenterPage(r *fastglue.Request, name string, data map[string]interface{}) error {
	app := r.Context.(*App)
	if err := app.tmpl.RenderWebPage(r.RequestCtx, name, data); err != nil {
		return err
	}
	r.RequestCtx.Response.Header.Set("Cache-Control", helpCenterCacheControl)
	r.RequestCtx.Response.Header.Del("Pragma")
	r.RequestCtx.Response.Header.Del("Expires")
	return nil
}

// sendXML writes v as an XML document.
func sendXML(r *fastglue.Request, v any) error {
	out, err := xml.Marshal(v)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	r.RequestCtx.SetContentType("application/xml; charset=utf-8")
	r.RequestCtx.Response.Header.Set("Cache-Control", helpCenterXMLCacheControl)
	fmt.Fprint(r.RequestCtx, xml.Header)
	r.RequestCtx.Write(out)
	return nil
}

// countArticleView records a view after the page is served, counting cache hits too.
func countArticleView(h fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		err := h(r)
		switch r.RequestCtx.Response.StatusCode() {
		case fasthttp.StatusOK, fasthttp.StatusNotModified:
			if !isCrawler(r) {
				app := r.Context.(*App)
				slug, _ := r.RequestCtx.UserValue("slug").(string)
				articleSlug, _ := r.RequestCtx.UserValue("article_slug").(string)
				locale, _ := r.RequestCtx.UserValue("locale").(string)
				app.helpcenter.IncrementPublishedArticleView(slug, strings.TrimSuffix(articleSlug, markdownSlugExtension), locale)
			}
		}
		return err
	}
}

func cachedHCPage(h fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return cacheHCPage(h, false)
}

func cachedHCNoIndexPage(h fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return cacheHCPage(h, true)
}

// cacheHCPage owns every response header: the cache restores only body and content type.
func cacheHCPage(h fastglue.FastRequestHandler, noIndex bool) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		app := r.Context.(*App)
		r.RequestCtx.SetUserValue(helpCenterCacheNamespaceKey, helpCenterCacheNamespace)
		miss := false
		rendered := func(r *fastglue.Request) error {
			miss = true
			return h(r)
		}
		err := app.fc.Cached(rendered, helpCenterCacheOpts, helpCenterCacheGroup)(r)
		switch r.RequestCtx.Response.StatusCode() {
		case fasthttp.StatusOK, fasthttp.StatusNotModified:
			r.RequestCtx.Response.Header.Set("Cache-Control", helpCenterCacheControl)
			r.RequestCtx.Response.Header.Del("Pragma")
			r.RequestCtx.Response.Header.Del("Expires")
			status := "HIT"
			if miss {
				status = "MISS"
			}
			r.RequestCtx.Response.Header.Set("X-Cache", status)
			r.RequestCtx.Response.Header.Set("X-Content-Type-Options", "nosniff")
			r.RequestCtx.Response.Header.Set("X-Frame-Options", "SAMEORIGIN")
			r.RequestCtx.Response.Header.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			if noIndex || isMarkdownRequest(r) {
				r.RequestCtx.Response.Header.Set("X-Robots-Tag", noIndexHeader)
			}
		}
		return err
	}
}

func isMarkdownRequest(r *fastglue.Request) bool {
	slug, _ := r.RequestCtx.UserValue("article_slug").(string)
	return strings.HasSuffix(slug, markdownSlugExtension)
}

func clearsHCCache(h fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		err := h(r)
		if r.RequestCtx.Response.StatusCode() < fasthttp.StatusMultipleChoices {
			app := r.Context.(*App)
			if err := app.fc.DelGroup(helpCenterCacheNamespace, helpCenterCacheGroup); err != nil {
				app.lo.Error("error clearing help center cache", "error", err)
			}
		}
		return err
	}
}

// isCrawler reports whether the request came from a bot rather than a reader; HEAD probes never count as readers.
func isCrawler(r *fastglue.Request) bool {
	return r.RequestCtx.IsHead() || crawlerUARe.Match(r.RequestCtx.Request.Header.UserAgent())
}

// helpCenterRootURL returns the app root URL without its trailing slash. Canonical URLs,
// sitemaps and robots.txt all read it from here so they can never disagree.
func helpCenterRootURL(app *App) string {
	return strings.TrimRight(app.consts.Load().(*constants).AppBaseURL, "/")
}

// helpCenterBaseURL returns the origin a reader sees this help center on: its custom domain when set, else the app root URL.
func helpCenterBaseURL(app *App, hc hcmodels.HelpCenter) string {
	if origin := helpCenterCustomOrigin(hc); origin != "" {
		return origin
	}
	return helpCenterRootURL(app)
}

// helpCenterCustomOrigin returns the origin of hc's custom domain, empty when unset or unparseable.
func helpCenterCustomOrigin(hc hcmodels.HelpCenter) string {
	if hc.CustomDomain == "" {
		return ""
	}
	return urlOrigin(hc.CustomDomain)
}

// isRootHost reports whether host is the app root URL's hostname.
func isRootHost(app *App, host string) bool {
	return strings.EqualFold(urlHostname(helpCenterRootURL(app)), host)
}

// helpCenterByHost returns the active help center whose custom domain hostname is host.
func helpCenterByHost(app *App, host string) (hcmodels.HelpCenter, bool) {
	if isRootHost(app, host) {
		return hcmodels.HelpCenter{}, false
	}
	helpCenters, err := app.helpcenter.GetActiveHelpCenters()
	if err != nil {
		return hcmodels.HelpCenter{}, false
	}
	for _, hc := range helpCenters {
		if helpCenterCustomOrigin(hc) != "" && strings.EqualFold(urlHostname(hc.CustomDomain), host) {
			return hc, true
		}
	}
	return hcmodels.HelpCenter{}, false
}

// helpCenterHostNotFound serves unmatched paths on a help center's custom domain by prepending /hc/{slug} and re-running the router.
func helpCenterHostNotFound(app *App, g *fastglue.Fastglue) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		if (ctx.IsGet() || ctx.IsHead()) && ctx.UserValue(hcHostRewriteKey) == nil {
			host := hostWithoutPort(string(ctx.Host()))
			// The custom-domain lookup below hits the DB; throttle it like every registered public route.
			if !isRootHost(app, host) {
				if app.rateLimit.Check(ctx, "public") != nil {
					return
				}
				ctx.SetUserValue(rateLimitPaidKey, "public")
			}
			if hc, ok := helpCenterByHost(app, host); ok {
				// Strip the trailing slash here: the router's own trailing-slash redirect would otherwise expose the rewritten /hc/{slug} path in its Location.
				if path := string(ctx.Path()); len(path) > 1 && strings.HasSuffix(path, "/") {
					uri := strings.TrimRight(path, "/")
					if uri == "" {
						uri = "/"
					}
					if qs := ctx.URI().QueryString(); len(qs) > 0 {
						uri += "?" + string(qs)
					}
					redirectPath(ctx, uri, fasthttp.StatusMovedPermanently)
					return
				}
				ctx.SetUserValue(hcHostRewriteKey, true)
				ctx.Request.SetRequestURI("/hc/" + hc.Slug + string(ctx.RequestURI()))
				g.Router.Handler(ctx)
				return
			}
		}
		fastglue.NotFoundHandler(ctx)
	}
}

// helpCenterHostHome redirects the custom-domain root to the help center's default-locale home.
func helpCenterHostHome(h fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		app := r.Context.(*App)
		host := hostWithoutPort(string(r.RequestCtx.Host()))
		if !isRootHost(app, host) {
			// The custom-domain lookup below hits the DB; throttle it like every registered public route.
			if app.rateLimit.Check(r.RequestCtx, "public") != nil {
				return nil
			}
			if hc, ok := helpCenterByHost(app, host); ok {
				uri := helpCenterHomePath(hc, hc.DefaultLocale)
				if qs := r.RequestCtx.URI().QueryString(); len(qs) > 0 {
					uri += "?" + string(qs)
				}
				redirectPath(r.RequestCtx, uri, fasthttp.StatusFound)
				return nil
			}
		}
		return h(r)
	}
}

// redirectHelpCenterCanonicalHost 301s a /hc/{slug} path to its custom-domain equivalent, unless the request arrived via the host rewrite.
func redirectHelpCenterCanonicalHost(r *fastglue.Request, hc hcmodels.HelpCenter) bool {
	origin := helpCenterCustomOrigin(hc)
	if origin == "" || r.RequestCtx.UserValue(hcHostRewriteKey) != nil {
		return false
	}
	uri := origin + strings.TrimPrefix(string(r.RequestCtx.Path()), "/hc/"+hc.Slug)
	if qs := r.RequestCtx.URI().QueryString(); len(qs) > 0 {
		uri += "?" + string(qs)
	}
	r.RequestCtx.Redirect(uri, fasthttp.StatusMovedPermanently)
	return true
}

// helpCentersForHost returns the help centers whose custom domain is this request's host, else the app-root ones.
func helpCentersForHost(r *fastglue.Request, helpCenters []hcmodels.HelpCenter) []hcmodels.HelpCenter {
	var (
		host       = hostWithoutPort(string(r.RequestCtx.Host()))
		matched    []hcmodels.HelpCenter
		rootHosted []hcmodels.HelpCenter
	)
	for _, hc := range helpCenters {
		if hc.CustomDomain == "" {
			rootHosted = append(rootHosted, hc)
			continue
		}
		if strings.EqualFold(urlHostname(hc.CustomDomain), host) {
			matched = append(matched, hc)
		}
	}
	if len(matched) > 0 {
		return matched
	}
	return rootHosted
}

// helpCenterHostRootURL returns the origin these help centers are served on.
func helpCenterHostRootURL(app *App, helpCenters []hcmodels.HelpCenter) string {
	for _, hc := range helpCenters {
		if hc.CustomDomain != "" {
			return urlOrigin(hc.CustomDomain)
		}
	}
	return helpCenterRootURL(app)
}

// urlHostname returns the URL's host without any port, so a public URL saved with an
// explicit port still matches the Host header and vice versa.
func urlHostname(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

func hostWithoutPort(h string) string {
	if host, _, err := net.SplitHostPort(h); err == nil {
		return host
	}
	return h
}

func urlOrigin(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

// publicAssetPaths rewrites stored absolute media URLs to root-relative ones so uploads resolve
// on whichever host serves the page.
func publicAssetPaths(app *App, s string) string {
	root := helpCenterRootURL(app)
	if root == "" {
		return s
	}
	return strings.ReplaceAll(s, root+media.PublicURI, media.PublicURI)
}

// absoluteURL resolves a root-relative URL against the app root URL, since social and
// structured-data consumers reject relative image URLs.
func absoluteURL(root, u string) string {
	if u == "" || strings.Contains(u, "://") || strings.HasPrefix(u, "//") {
		return u
	}
	return root + "/" + strings.TrimLeft(u, "/")
}

// helpCenterPathPrefix returns the path prefix hc's pages live under: none on a custom domain, /hc/{slug} on the app host.
func helpCenterPathPrefix(hc hcmodels.HelpCenter) string {
	if helpCenterCustomOrigin(hc) != "" {
		return ""
	}
	return "/hc/" + hc.Slug
}

func helpCenterHomePath(hc hcmodels.HelpCenter, locale string) string {
	if locale == "" {
		locale = hc.DefaultLocale
	}
	return fmt.Sprintf("%s/%s", helpCenterPathPrefix(hc), locale)
}

func collectionPath(hc hcmodels.HelpCenter, locale, collectionSlug string) string {
	return fmt.Sprintf("%s/%s/collections/%s", helpCenterPathPrefix(hc), locale, collectionSlug)
}

func articlePath(hc hcmodels.HelpCenter, locale, articleSlug string) string {
	return fmt.Sprintf("%s/%s/articles/%s", helpCenterPathPrefix(hc), locale, articleSlug)
}

func searchPath(hc hcmodels.HelpCenter, locale string) string {
	return fmt.Sprintf("%s/%s/search", helpCenterPathPrefix(hc), locale)
}

// helpCenterAlternates returns the hreflang set for a page: only the locales the page
// actually exists in, so no alternate points at a 404. Empty for single-locale help centers.
func helpCenterAlternates(hc hcmodels.HelpCenter, translated []string, pathFor func(string) string) []localeLink {
	locales := helpCenterLocales(hc)
	if len(locales) < 2 {
		return nil
	}
	links := make([]localeLink, 0, len(locales))
	for _, loc := range locales {
		if slices.Contains(translated, loc) {
			links = append(links, localeLink{Locale: loc, Path: pathFor(loc)})
		}
	}
	return links
}

// helpCenterLocaleLinks returns where the language switcher sends the reader for each locale:
// the same page when it exists there, the locale home otherwise.
func helpCenterLocaleLinks(hc hcmodels.HelpCenter, translated []string, pathFor func(string) string) []localeLink {
	locales := helpCenterLocales(hc)
	links := make([]localeLink, 0, len(locales))
	for _, loc := range locales {
		path := helpCenterHomePath(hc, loc)
		if slices.Contains(translated, loc) {
			path = pathFor(loc)
		}
		links = append(links, localeLink{Locale: loc, Path: path})
	}
	return links
}

// defaultLocalePath returns the x-default hreflang target, empty when the page has no
// default-locale version.
func defaultLocalePath(hc hcmodels.HelpCenter, translated []string, pathFor func(string) string) string {
	if len(helpCenterLocales(hc)) < 2 || !slices.Contains(translated, hc.DefaultLocale) {
		return ""
	}
	return pathFor(hc.DefaultLocale)
}

// homeJSONLD returns the WebSite structured data for a help center home page, including the
// sitelinks search box target.
func homeJSONLD(root string, hc hcmodels.HelpCenter, locale string) template.JS {
	home := root + helpCenterHomePath(hc, locale)
	site := map[string]any{
		"@context":   schemaOrgContext,
		"@type":      "WebSite",
		"name":       hc.Name,
		"url":        home,
		"inLanguage": locale,
		"potentialAction": map[string]any{
			"@type": "SearchAction",
			"target": map[string]any{
				"@type":       "EntryPoint",
				"urlTemplate": root + searchPath(hc, locale) + "?q={search_term_string}",
			},
			"query-input": "required name=search_term_string",
		},
	}
	if d := firstNonEmpty(hc.MetaDescription, helpCenterTheme(hc).Header.Heading); d != "" {
		site["description"] = d
	}
	return jsonLD([]any{site})
}

// collectionJSONLD returns the CollectionPage and breadcrumb structured data for a collection page.
func collectionJSONLD(root string, hc hcmodels.HelpCenter, collection hcmodels.TreeCollection, locale, canonicalPath string) template.JS {
	page := map[string]any{
		"@context":   schemaOrgContext,
		"@type":      "CollectionPage",
		"name":       collection.Name,
		"url":        root + canonicalPath,
		"inLanguage": locale,
		"isPartOf":   map[string]any{"@type": "WebSite", "name": hc.Name, "url": root + helpCenterHomePath(hc, locale)},
	}
	if collection.Description != "" {
		page["description"] = collection.Description
	}
	crumbs := breadcrumbJSONLD(root, hc, locale, []localeLink{{Locale: collection.Name, Path: canonicalPath}})
	return jsonLD([]any{page, crumbs})
}

// articleJSONLD returns the Article and breadcrumb structured data for an article page.
func articleJSONLD(root string, hc hcmodels.HelpCenter, collection hcmodels.Collection, article hcmodels.Article, locale, canonicalPath, image string) template.JS {
	art := map[string]any{
		"@context":         schemaOrgContext,
		"@type":            "Article",
		"headline":         article.Title,
		"url":              root + canonicalPath,
		"inLanguage":       locale,
		"datePublished":    article.CreatedAt.Format(time.RFC3339),
		"dateModified":     article.UpdatedAt.Format(time.RFC3339),
		"mainEntityOfPage": map[string]any{"@type": "WebPage", "@id": root + canonicalPath},
		"publisher":        map[string]any{"@type": "Organization", "name": hc.Name},
	}
	if d := firstNonEmpty(article.MetaDescription, article.Excerpt); d != "" {
		art["description"] = d
	}
	if image != "" {
		art["image"] = image
	}
	if article.AuthorName != nil && strings.TrimSpace(*article.AuthorName) != "" {
		art["author"] = map[string]any{"@type": "Person", "name": strings.TrimSpace(*article.AuthorName)}
	} else {
		art["author"] = map[string]any{"@type": "Organization", "name": hc.Name}
	}

	trail := []localeLink{}
	if collection.Name != "" {
		trail = append(trail, localeLink{Locale: collection.Name, Path: collectionPath(hc, locale, collection.Slug)})
	}
	trail = append(trail, localeLink{Locale: article.Title, Path: canonicalPath})
	return jsonLD([]any{art, breadcrumbJSONLD(root, hc, locale, trail)})
}

// breadcrumbJSONLD builds a BreadcrumbList rooted at the help center home, where each trail
// entry carries its label in Locale and its path in Path.
func breadcrumbJSONLD(root string, hc hcmodels.HelpCenter, locale string, trail []localeLink) map[string]any {
	items := []any{map[string]any{
		"@type":    "ListItem",
		"position": 1,
		"name":     hc.Name,
		"item":     root + helpCenterHomePath(hc, locale),
	}}
	for i, t := range trail {
		items = append(items, map[string]any{
			"@type":    "ListItem",
			"position": i + 2,
			"name":     t.Locale,
			"item":     root + t.Path,
		})
	}
	return map[string]any{
		"@context":        schemaOrgContext,
		"@type":           "BreadcrumbList",
		"itemListElement": items,
	}
}

// jsonLD marshals structured data for a ld+json script tag. encoding/json escapes the HTML
// delimiters, so a title can't close the tag.
func jsonLD(v any) template.JS {
	b, err := json.Marshal(v)
	if err != nil {
		return template.JS("")
	}
	return template.JS(b)
}

// authorInitial returns the first letter of the article author's name for the avatar fallback.
func authorInitial(article hcmodels.Article) string {
	if article.AuthorName == nil {
		return ""
	}
	runes := []rune(strings.TrimSpace(*article.AuthorName))
	if len(runes) == 0 {
		return ""
	}
	return string(runes[0])
}

// firstNonEmpty returns the first value that isn't blank.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// flattenCollections returns every collection in the tree, depth first.
func flattenCollections(cols []hcmodels.TreeCollection, out []hcmodels.TreeCollection) []hcmodels.TreeCollection {
	for _, c := range cols {
		out = append(out, c)
		out = flattenCollections(c.Children, out)
	}
	return out
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

// resolveLocale returns the locale path segment and whether the help center serves it,
// falling back to the help center's default when the segment is absent.
func resolveLocale(r *fastglue.Request, hc hcmodels.HelpCenter) (string, bool) {
	v, _ := r.RequestCtx.UserValue("locale").(string)
	loc := strings.TrimSpace(v)
	if loc == "" || loc == hc.DefaultLocale {
		return hc.DefaultLocale, true
	}
	return loc, slices.Contains(helpCenterLocales(hc), loc)
}

// resolveQueryLocale returns the "locale" query arg when the help center serves it, else its default.
func resolveQueryLocale(r *fastglue.Request, hc hcmodels.HelpCenter) string {
	loc := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("locale")))
	if loc == "" || !slices.Contains(helpCenterLocales(hc), loc) {
		return hc.DefaultLocale
	}
	return loc
}

// helpCenterLocales returns the help center's configured locale codes.
func helpCenterLocales(hc hcmodels.HelpCenter) []string {
	locales := []string{}
	if len(hc.AllowedLocales) > 0 {
		if err := json.Unmarshal(hc.AllowedLocales, &locales); err != nil {
			return nil
		}
	}
	return locales
}

func helpCenterTheme(hc hcmodels.HelpCenter) hcmodels.Theme {
	theme := hcmodels.DefaultTheme()
	if len(hc.Theme) > 0 {
		if err := json.Unmarshal(hc.Theme, &theme); err != nil {
			return hcmodels.DefaultTheme()
		}
	}
	return theme
}

func hideArticleAuthor(theme hcmodels.Theme, article *hcmodels.Article) {
	if theme.Article.ShowAuthor {
		return
	}
	article.AuthorName = nil
	article.AuthorAvatar = nil
}

func hideTreeAuthors(theme hcmodels.Theme, cols []hcmodels.TreeCollection) {
	for i := range cols {
		if !theme.Cards.ShowAuthors {
			cols[i].Authors = []hcmodels.ArticleAuthor{}
			cols[i].AuthorCount = 0
		}
		for j := range cols[i].Articles {
			hideArticleAuthor(theme, &cols[i].Articles[j])
		}
		hideTreeAuthors(theme, cols[i].Children)
	}
}

// hcPageName returns the page's template name under the help center's chosen page template.
func hcPageName(hc hcmodels.HelpCenter, page string) string {
	return firstNonEmpty(hc.Template, hcmodels.TemplateClassic) + "-" + page
}

// sidebarTree returns the published tree for page templates that render a navigation sidebar on every page.
func sidebarTree(app *App, hc hcmodels.HelpCenter, locale string) []hcmodels.TreeCollection {
	if hc.Template != hcmodels.TemplateDocs {
		return nil
	}
	tree, err := app.helpcenter.GetPublicTree(hc, locale)
	if err != nil {
		return nil
	}
	return tree.Tree
}

// helpCenterTemplateData shapes a help center row for the public templates.
func helpCenterTemplateData(app *App, hc hcmodels.HelpCenter, locale string) map[string]interface{} {
	theme := helpCenterTheme(hc)
	theme.Favicon = publicAssetPaths(app, theme.Favicon)
	theme.Header.BackgroundImage = publicAssetPaths(app, theme.Header.BackgroundImage)
	pageTemplate := hc.Template
	if pageTemplate == "" {
		pageTemplate = hcmodels.TemplateClassic
	}
	return map[string]interface{}{
		"Slug":              hc.Slug,
		"Name":              hc.Name,
		"Template":          pageTemplate,
		"BaseURL":           helpCenterBaseURL(app, hc),
		"BasePath":          helpCenterHomePath(hc, locale),
		"PageTitle":         hc.PageTitle,
		"HeaderText":        theme.Header.Heading,
		"LogoURL":           publicAssetPaths(app, theme.LogoURL),
		"Color":             theme.Color,
		"DefaultLocale":     hc.DefaultLocale,
		"CurrentLocale":     locale,
		"OGLocale":          strings.ReplaceAll(locale, "-", "_"),
		"Dir":               localeDir(locale),
		"AvailableLocales":  helpCenterLocales(hc),
		"NavLinks":          theme.NavLinks,
		"Theme":             theme,
		"ThemeCSS":          buildThemeCSSVars(theme),
		"AnnouncementKey":   announcementKey(hc.Slug, theme.Announcement),
		"TaglineHTML":       template.HTML(helpcenter.RenderInlineMarkdown(theme.Tagline)),
		"FooterTaglineHTML": template.HTML(helpcenter.RenderInlineMarkdown(theme.Footer.Tagline)),
		"AnnouncementHTML":  template.HTML(helpcenter.RenderInlineMarkdown(theme.Announcement.Text)),
		"CustomCSS":         template.CSS(hc.CustomCSS),
		"CustomJS":          template.JS(hc.CustomJS),
	}
}

// announcementKey keys the dismissal on help center and content, so an edited announcement reappears for visitors who dismissed the old one.
func announcementKey(hcSlug string, a hcmodels.AnnouncementTheme) string {
	if a.Text == "" {
		return ""
	}
	h := fnv.New32a()
	h.Write([]byte(hcSlug + "|" + a.Text + "|" + a.LinkLabel + "|" + a.LinkURL))
	return fmt.Sprintf("%x", h.Sum32())
}

// buildThemeCSSVars emits the theme's CSS custom-property overrides.
func buildThemeCSSVars(t hcmodels.Theme) template.CSS {
	var b strings.Builder
	switch t.Header.BackgroundType {
	case "image":
		if t.Header.BackgroundImage != "" {
			fmt.Fprintf(&b, "--hc-header-img:url(%s);", t.Header.BackgroundImage)
			b.WriteString("--hc-hero-text-shadow:0 1px 2px rgba(0,0,0,.35),0 2px 16px rgba(0,0,0,.25);")
			if t.Header.TextColor == "" {
				b.WriteString("--hc-header-text:#ffffff;--hc-header-scrim:linear-gradient(rgba(10,12,18,.45),rgba(10,12,18,.45));")
			}
		}
	case "gradient":
		if t.Header.GradientFrom != "" && t.Header.GradientTo != "" {
			fmt.Fprintf(&b, "--hc-header-bg:linear-gradient(180deg,%s,%s);", t.Header.GradientFrom, t.Header.GradientTo)
			if t.Header.TextColor == "" {
				fmt.Fprintf(&b, "--hc-header-text:%s;", readableOn(t.Header.GradientFrom, t.Header.GradientTo))
			}
		}
	case "solid":
		if t.Header.BackgroundColor != "" {
			fmt.Fprintf(&b, "--hc-header-bg:%s;", t.Header.BackgroundColor)
			if t.Header.TextColor == "" {
				fmt.Fprintf(&b, "--hc-header-text:%s;", readableOn(t.Header.BackgroundColor))
			}
		}
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

// renderHelpCenterNotFound renders the help center's themed 404, falling back to the
// generic error page when the help center is nil.
func renderHelpCenterNotFound(r *fastglue.Request, hc *hcmodels.HelpCenter) error {
	return renderHelpCenterStatusPage(r, hc, fasthttp.StatusNotFound)
}

// renderHelpCenterPageError renders the themed 404 for missing pages and a themed 500 for everything else.
func renderHelpCenterPageError(r *fastglue.Request, hc *hcmodels.HelpCenter, err error) error {
	if e, ok := err.(envelope.Error); ok && e.ErrorType == envelope.NotFoundError {
		return renderHelpCenterNotFound(r, hc)
	}
	return renderHelpCenterStatusPage(r, hc, fasthttp.StatusInternalServerError)
}

func renderHelpCenterStatusPage(r *fastglue.Request, hc *hcmodels.HelpCenter, status int) error {
	app := r.Context.(*App)
	headingKey, textKey := "globals.messages.pageNotFound", "helpCenter.notFoundText"
	if status == fasthttp.StatusInternalServerError {
		headingKey, textKey = "globals.messages.somethingWentWrong", "helpCenter.errorText"
	}
	if hc != nil {
		helpCenter := *hc
		locale, ok := resolveLocale(r, helpCenter)
		if !ok {
			locale = helpCenter.DefaultLocale
		}
		data := helpCenterTemplateData(app, helpCenter, locale)
		lcl := localeI18n(app, locale)
		r.RequestCtx.Response.Header.Set("X-Robots-Tag", noIndexHeader)
		rerr := app.tmpl.RenderWebPage(r.RequestCtx, hcPageName(helpCenter, "help-notfound"), map[string]interface{}{
			"L": lcl,
			"Data": map[string]interface{}{
				"Title":       lcl.T(headingKey),
				"NoIndex":     true,
				"ErrorCode":   strconv.Itoa(status),
				"ErrorTitle":  lcl.T(headingKey),
				"ErrorText":   lcl.T(textKey),
				"LocaleLinks": helpCenterLocaleLinks(helpCenter, nil, func(l string) string { return helpCenterHomePath(helpCenter, l) }),
				"HelpCenter":  data,
				"Tree":        sidebarTree(app, helpCenter, locale),
			},
		})
		r.RequestCtx.SetStatusCode(status)
		return rerr
	}
	rerr := app.tmpl.RenderWebPage(r.RequestCtx, "error", map[string]interface{}{
		"Data": map[string]interface{}{
			"Title":        app.i18n.T(headingKey),
			"ErrorMessage": app.i18n.T(headingKey),
		},
	})
	r.RequestCtx.SetStatusCode(status)
	return rerr
}

func validateHelpCenter(app *App, req *helpcenter.HelpCenterRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	req.Slug = strings.TrimSpace(req.Slug)
	req.PageTitle = strings.TrimSpace(req.PageTitle)
	req.MetaDescription = strings.TrimSpace(req.MetaDescription)
	if req.Name == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "`name`"), nil)
	}
	if req.Slug == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "`slug`"), nil)
	}
	if req.PageTitle == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "`page_title`"), nil)
	}
	req.Theme = json.RawMessage(publicAssetPaths(app, string(req.Theme)))
	return nil
}

func validateCollection(app *App, req *helpcenter.CollectionRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "`name`"), nil)
	}
	return nil
}

func validateArticle(app *App, req *helpcenter.ArticleRequest) error {
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "`title`"), nil)
	}
	if strings.TrimSpace(req.Content) == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "`content`"), nil)
	}
	req.Content = publicAssetPaths(app, req.Content)
	req.MetaImageURL = publicAssetPaths(app, req.MetaImageURL)
	return nil
}

// loadLucideIcons parses the vendored lucide sprite into ready-to-inline SVG elements keyed by icon name.
func loadLucideIcons(fs stuffbin.FileSystem) map[string]template.HTML {
	icons := map[string]template.HTML{}
	b, err := fs.Read(lucideSpritePath)
	if err != nil {
		log.Printf("error reading lucide sprite: %v", err)
		return icons
	}
	for _, m := range lucideSymbolRe.FindAllSubmatch(b, -1) {
		icons[string(m[1])] = template.HTML(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">` + string(m[2]) + `</svg>`)
	}
	return icons
}

// readableOn returns the dark or light header text color, whichever holds the higher worst-case
// contrast across the given hex backgrounds. A gradient passes both of its endpoints.
func readableOn(hexColors ...string) string {
	darkL, _ := relativeLuminance(headerTextDark)
	darkWorst, lightWorst := math.Inf(1), math.Inf(1)
	for _, hexColor := range hexColors {
		bg, ok := relativeLuminance(hexColor)
		if !ok {
			return headerTextDark
		}
		darkWorst = min(darkWorst, contrastRatio(bg, darkL))
		lightWorst = min(lightWorst, contrastRatio(bg, 1))
	}
	if lightWorst > darkWorst {
		return headerTextLight
	}
	return headerTextDark
}

// relativeLuminance returns the WCAG relative luminance of any hex color hexColorRe accepts; alpha is dropped.
func relativeLuminance(hexColor string) (float64, bool) {
	c := strings.TrimPrefix(strings.TrimSpace(hexColor), "#")
	if len(c) == 3 || len(c) == 4 {
		c = string([]byte{c[0], c[0], c[1], c[1], c[2], c[2]})
	}
	if len(c) == 8 {
		c = c[:6]
	}
	if len(c) != 6 {
		return 0, false
	}
	v, err := strconv.ParseUint(c, 16, 32)
	if err != nil {
		return 0, false
	}
	toLinear := func(ch float64) float64 {
		ch /= 255
		if ch <= 0.03928 {
			return ch / 12.92
		}
		return math.Pow((ch+0.055)/1.055, 2.4)
	}
	r, g, b := float64((v>>16)&0xff), float64((v>>8)&0xff), float64(v&0xff)
	return 0.2126*toLinear(r) + 0.7152*toLinear(g) + 0.0722*toLinear(b), true
}

func contrastRatio(a, b float64) float64 {
	return (max(a, b) + 0.05) / (min(a, b) + 0.05)
}

// localeDir returns the text direction for a locale tag such as "ar" or "ar-EG".
func localeDir(locale string) string {
	base, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(locale)), "-")
	if slices.Contains(rtlLanguages, base) {
		return "rtl"
	}
	return "ltr"
}

// renderHelpCenterArticlePreview renders a sample article page from unsaved settings; TOC and related list are built server-side because the preview iframe is sandboxed without scripts.
func renderHelpCenterArticlePreview(r *fastglue.Request, helpCenter hcmodels.HelpCenter, locale string) error {
	var (
		app  = r.Context.(*App)
		i18n = localeI18n(app, locale)
		body = template.HTMLEscapeString(i18n.T("helpCenter.preview.sampleArticleBody"))
		item = template.HTMLEscapeString(i18n.T("helpCenter.preview.sampleArticleListItem"))
		toc  = []previewTOCItem{
			{ID: "sample-1", Title: i18n.T("helpCenter.preview.sampleArticleHeading1")},
			{ID: "sample-2", Title: i18n.T("helpCenter.preview.sampleArticleHeading2")},
			{ID: "sample-3", Title: i18n.T("helpCenter.preview.sampleArticleHeading3")},
		}
		author  = i18n.T("helpCenter.preview.sampleAuthor")
		article = hcmodels.Article{
			Slug:       "sample-article",
			Locale:     locale,
			AuthorName: &author,
			Title:      i18n.T("helpCenter.preview.sampleArticleTitle"),
			Excerpt:    i18n.T("helpCenter.preview.sampleArticleExcerpt"),
			UpdatedAt:  time.Now(),
			Content: fmt.Sprintf(
				`<h2 id="%s">%s</h2><p>%s</p><h2 id="%s">%s</h2><p>%s</p><ul><li>%s</li><li>%s</li></ul><h2 id="%s">%s</h2><p>%s</p>`,
				toc[0].ID, template.HTMLEscapeString(toc[0].Title), body,
				toc[1].ID, template.HTMLEscapeString(toc[1].Title), body, item, item,
				toc[2].ID, template.HTMLEscapeString(toc[2].Title), body),
		}
		collection = hcmodels.Collection{
			Slug: "sample-collection",
			Name: i18n.T("helpCenter.preview.sampleCollection"),
		}
		related = []hcmodels.Article{
			{Slug: "sample-1", Title: toc[0].Title},
			{Slug: "sample-2", Title: toc[1].Title},
			{Slug: "sample-3", Title: toc[2].Title},
		}
	)
	if err := app.tmpl.RenderWebPage(r.RequestCtx, hcPageName(helpCenter, "help-article"), map[string]interface{}{
		"L": i18n,
		"Data": map[string]interface{}{
			"Title":         article.Title,
			"ModifiedTime":  article.UpdatedAt.Format(time.RFC3339),
			"HelpCenter":    helpCenterTemplateData(app, helpCenter, locale),
			"Article":       article,
			"AuthorInitial": authorInitial(article),
			"Collection":    collection,
			"Related":       related,
			"TOC":           toc,
			"Tree":          sidebarTree(app, helpCenter, locale),
			"Content":       template.HTML(article.Content),
		},
	}); err != nil {
		return sendErrorEnvelope(r, err)
	}
	r.RequestCtx.Response.Header.Set("Cache-Control", "no-store")
	return nil
}

// redirectPath sends a path-only Location; ctx.Redirect would absolutize it to http:// behind a TLS-terminating proxy.
// The URI round-trip is what escapes the path: a bare "/\host" Location is read as an authority by browsers.
func redirectPath(ctx *fasthttp.RequestCtx, uri string, statusCode int) {
	u := fasthttp.AcquireURI()
	defer fasthttp.ReleaseURI(u)
	u.Update(uri)
	ctx.Response.Header.SetCanonical([]byte(fasthttp.HeaderLocation), u.RequestURI())
	ctx.SetStatusCode(statusCode)
	ctx.Response.SetBodyString("")
}
