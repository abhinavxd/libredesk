package main

import (
	"strconv"

	"github.com/zerodha/fastglue"
)

// handleShowCSAT renders the CSAT page for a given csat.
func handleShowCSAT(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("uuid").(string)
	)

	csat, err := app.csat.Get(uuid)
	if err != nil {
		return app.tmpl.RenderWebPage(r.RequestCtx, "error", map[string]interface{}{
			"Data": map[string]interface{}{
				"ErrorMessage": app.i18n.T("globals.messages.pageNotFound"),
			},
		})
	}

	if csat.ResponseTimestamp.Valid {
		return app.tmpl.RenderWebPage(r.RequestCtx, "info", map[string]interface{}{
			"Data": map[string]interface{}{
				"Title":   app.i18n.T("globals.messages.thankYou"),
				"Message": app.i18n.T("csat.thankYouMessage"),
			},
		})
	}

	conversation, err := app.conversation.GetConversation(csat.ConversationID, "")
	if err != nil {
		return app.tmpl.RenderWebPage(r.RequestCtx, "error", map[string]interface{}{
			"Data": map[string]interface{}{
				"ErrorMessage": app.i18n.T("globals.messages.pageNotFound"),
			},
		})
	}

	return app.tmpl.RenderWebPage(r.RequestCtx, "csat", map[string]interface{}{
		"Data": map[string]interface{}{
			"Title": app.i18n.T("csat.pageTitle"),
			"CSAT": map[string]interface{}{
				"UUID": csat.UUID,
			},
			"Conversation": map[string]interface{}{
				"Subject":         conversation.Subject.String,
				"ReferenceNumber": conversation.ReferenceNumber,
			},
		},
	})
}

// handleUpdateCSATResponse updates the CSAT response for a given csat.
func handleUpdateCSATResponse(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		uuid     = r.RequestCtx.UserValue("uuid").(string)
		rating   = r.RequestCtx.FormValue("rating")
		feedback = string(r.RequestCtx.FormValue("feedback"))
	)

	ratingI, err := strconv.Atoi(string(rating))
	if err != nil {
		return app.tmpl.RenderWebPage(r.RequestCtx, "error", map[string]interface{}{
			"Data": map[string]interface{}{
				"ErrorMessage": app.i18n.T("globals.messages.somethingWentWrong"),
			},
		})
	}

	if ratingI < 1 || ratingI > 5 {
		return app.tmpl.RenderWebPage(r.RequestCtx, "error", map[string]interface{}{
			"Data": map[string]interface{}{
				"ErrorMessage": app.i18n.T("globals.messages.somethingWentWrong"),
			},
		})
	}

	if uuid == "" {
		return app.tmpl.RenderWebPage(r.RequestCtx, "error", map[string]interface{}{
			"Data": map[string]interface{}{
				"ErrorMessage": app.i18n.T("globals.messages.somethingWentWrong"),
			},
		})
	}

	if err := app.csat.UpdateResponse(uuid, ratingI, feedback); err != nil {
		return app.tmpl.RenderWebPage(r.RequestCtx, "error", map[string]interface{}{
			"Data": map[string]interface{}{
				"ErrorMessage": err.Error(),
			},
		})
	}

	return app.tmpl.RenderWebPage(r.RequestCtx, "info", map[string]interface{}{
		"Data": map[string]interface{}{
			"Title":   app.i18n.T("csat.thankYou"),
			"Message": app.i18n.T("csat.thankYouMessage"),
		},
	})
}
