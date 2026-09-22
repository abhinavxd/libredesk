package main

import (
	"slices"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	hcmodels "github.com/abhinavxd/libredesk/internal/helpcenter/models"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/zerodha/fastglue"
)

func optionalWidgetAuth(next func(*fastglue.Request) error) func(*fastglue.Request) error {
	return func(r *fastglue.Request) error {
		if len(r.RequestCtx.Request.Header.Peek("Authorization")) == 0 {
			return validateWidgetInbox(next)(r)
		}
		return widgetAuth(next)(r)
	}
}

func handleWidgetHelp(r *fastglue.Request) error {
	app := r.Context.(*App)
	helpCenter, audience, err := widgetHelpCenter(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if helpCenter.ID == 0 {
		return r.SendEnvelope(nil)
	}
	locale := resolveQueryLocale(r, helpCenter)
	tree, err := app.helpcenter.GetPublicTree(helpCenter, locale)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if len(tree.Tree) == 0 && locale != helpCenter.DefaultLocale {
		locale = helpCenter.DefaultLocale
		tree, err = app.helpcenter.GetPublicTree(helpCenter, locale)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
	}
	hideTreeAuthors(helpCenterTheme(helpCenter), tree.Tree)
	popular, err := app.helpcenter.GetPopularArticles(helpCenter.Slug, locale, 5)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	for i := range popular {
		hideArticleAuthor(helpCenterTheme(helpCenter), &popular[i])
	}
	config, _ := getWidgetConfig(r)
	return r.SendEnvelope(map[string]any{
		"slug": helpCenter.Slug, "name": helpCenter.Name, "locale": locale,
		"url":  helpCenterBaseURL(app, helpCenter) + helpCenterPathPrefix(helpCenter),
		"tree": tree.Tree, "popular": popular, "audience": audience, "locales": helpCenterLocales(helpCenter),
		"featured_ids": config.Help.FeaturedIDs,
	})
}

func handleWidgetHelpSearch(r *fastglue.Request) error {
	app := r.Context.(*App)
	helpCenter, _, err := widgetHelpCenter(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if helpCenter.ID == 0 {
		return r.SendEnvelope([]hcmodels.Article{})
	}
	query := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("q")))
	if query == "" {
		return r.SendEnvelope([]hcmodels.Article{})
	}
	articles, err := app.helpcenter.SearchPublishedArticles(helpCenter.Slug, query, resolveQueryLocale(r, helpCenter), publicSearchLimit)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	app.helpcenter.LogSearch(helpCenter.ID, query, len(articles))
	for i := range articles {
		hideArticleAuthor(helpCenterTheme(helpCenter), &articles[i])
	}
	return r.SendEnvelope(articles)
}

func handleWidgetHelpArticle(r *fastglue.Request) error {
	app := r.Context.(*App)
	helpCenter, _, err := widgetHelpCenter(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if helpCenter.ID == 0 {
		return sendErrorEnvelope(r, envelope.NewError(envelope.NotFoundError, app.i18n.T("globals.messages.notFound"), nil))
	}
	slug := r.RequestCtx.UserValue("article_slug").(string)
	locale := resolveQueryLocale(r, helpCenter)
	article, err := app.helpcenter.GetPublishedArticle(helpCenter.Slug, slug, locale)
	if envErr, ok := err.(envelope.Error); ok && envErr.ErrorType == envelope.NotFoundError && locale != helpCenter.DefaultLocale {
		article, err = app.helpcenter.GetPublishedArticle(helpCenter.Slug, slug, helpCenter.DefaultLocale)
	}
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	hideArticleAuthor(helpCenterTheme(helpCenter), &article)
	translations, err := app.helpcenter.GetPublishedArticleTranslations(helpCenter.Slug, article.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	app.helpcenter.IncrementArticleViewCount(article.ID)
	return r.SendEnvelope(helpArticleResponse{Article: article, Translations: translations})
}

func widgetHelpCenter(r *fastglue.Request) (hcmodels.HelpCenter, livechat.HelpAudience, error) {
	app := r.Context.(*App)
	config, err := getWidgetConfig(r)
	if err != nil {
		return hcmodels.HelpCenter{}, livechat.HelpAudience{}, err
	}
	audience := config.Help.Visitors
	if !getWidgetIsVisitor(r) {
		audience = config.Help.Users
	}
	if config.Help.HelpCenterID == 0 || (!audience.Tab && !hasHelpHomeApp(config)) {
		return hcmodels.HelpCenter{}, audience, nil
	}
	helpCenter, err := app.helpcenter.GetHelpCenterByID(config.Help.HelpCenterID)
	if err != nil {
		return hcmodels.HelpCenter{}, audience, err
	}
	if !helpCenter.IsActive {
		return hcmodels.HelpCenter{}, audience, nil
	}
	return helpCenter, audience, nil
}

func widgetContact(r *fastglue.Request) (umodels.User, error) {
	id, err := getWidgetContactID(r)
	if err != nil {
		return umodels.User{}, nil
	}
	return r.Context.(*App).user.GetContactOrVisitor(id, "")
}

func hasHelpHomeApp(config livechat.Config) bool {
	return slices.ContainsFunc(config.HomeApps, func(app livechat.HomeApp) bool {
		return app.Type == livechat.HomeAppHelp
	})
}
