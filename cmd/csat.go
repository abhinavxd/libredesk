package main

import (
	"strconv"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type csatResponse struct {
	Rating   int    `json:"rating"`
	Feedback string `json:"feedback"`
}
const (
	maxCsatFeedbackLength = 1000
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

	conversation, err := app.conversation.GetConversation(csat.ConversationID, "", "")
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

	if ratingI < 0 || ratingI > 5 {
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

	// Trim feedback if it exceeds max length
	if len(feedback) > maxCsatFeedbackLength {
		feedback = feedback[:maxCsatFeedbackLength]
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
			"Title":   app.i18n.T("globals.messages.thankYou"),
			"Message": app.i18n.T("csat.thankYouMessage"),
		},
	})
}

// handleSubmitCSATResponse handles CSAT response submission from the widget API.
func handleSubmitCSATResponse(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("uuid").(string)
		req  = csatResponse{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid JSON", nil, envelope.InputError)
	}

	if req.Rating < 0 || req.Rating > 5 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Rating must be between 0 and 5 (0 means no rating)", nil, envelope.InputError)
	}

	// At least one of rating or feedback must be provided
	if req.Rating == 0 && req.Feedback == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Either rating or feedback must be provided", nil, envelope.InputError)
	}

	if uuid == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid UUID", nil, envelope.InputError)
	}

	// Update CSAT response
	if err := app.csat.UpdateResponse(uuid, req.Rating, req.Feedback); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}
