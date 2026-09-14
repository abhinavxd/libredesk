package main

import (
	"encoding/json"
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
	// maxCsatBodyBytes caps the request body on the public CSAT submit
	// endpoints: ~2x the largest legitimate submission (rating + a
	// 1000-char feedback, worst-case ~12KB when URL-encoded).
	maxCsatBodyBytes = 24 * 1024

	maxCsatFeedbackLength = 1000
	maxCsatMetaKeys       = 100
	maxCsatMetaKeyLength  = 100
	maxCsatMetaValLength  = 1000
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
		app  = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("uuid").(string)
	)

	rating, feedback, metaJSON, errKey := validateCSATForm(r)
	if errKey != "" {
		return app.tmpl.RenderWebPage(r.RequestCtx, "error", map[string]interface{}{
			"Data": map[string]interface{}{
				"ErrorMessage": app.i18n.T(errKey),
			},
		})
	}

	if err := app.csat.UpdateResponse(uuid, rating, feedback, metaJSON); err != nil {
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

// handleShowCSATWidget renders a minimal CSAT widget page (just stars) for iframe embedding.
func handleShowCSATWidget(r *fastglue.Request) error {
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

	return app.tmpl.RenderWebPage(r.RequestCtx, "csat-widget", map[string]interface{}{
		"Data": map[string]interface{}{
			"CSAT": map[string]interface{}{
				"UUID":      csat.UUID,
				"Responded": csat.ResponseTimestamp.Valid,
			},
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

	// Trim feedback if it exceeds max length.
	if len(req.Feedback) > maxCsatFeedbackLength {
		req.Feedback = req.Feedback[:maxCsatFeedbackLength]
	}

	// Update CSAT response
	if err := app.csat.UpdateResponse(uuid, req.Rating, req.Feedback, nil); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// validateCSATForm parses and validates the CSAT form submission.
// Returns rating (0 if not provided), trimmed feedback, meta JSON, and error message key if invalid.
func validateCSATForm(r *fastglue.Request) (int, string, json.RawMessage, string) {
	var (
		feedback = string(r.RequestCtx.FormValue("feedback"))
		rating   int
	)

	// Rating is optional (0 = not provided). If provided, must be 1-5.
	if rs := string(r.RequestCtx.FormValue("rating")); rs != "" {
		v, err := strconv.Atoi(rs)
		if err != nil || v < 1 || v > 5 {
			return 0, "", nil, "globals.messages.somethingWentWrong"
		}
		rating = v
	}

	// At least one of rating or feedback must be provided.
	if rating == 0 && feedback == "" {
		return 0, "", nil, "csat.pleaseFillRequired"
	}

	if len(feedback) > maxCsatFeedbackLength {
		feedback = feedback[:maxCsatFeedbackLength]
	}

	// Collect extra form fields into meta, skipping the known fields.
	meta := make(map[string]string)
	r.RequestCtx.PostArgs().VisitAll(func(key, value []byte) {
		k := string(key)
		if k == "rating" || k == "feedback" {
			return
		}
		if len(meta) >= maxCsatMetaKeys {
			return
		}
		if len(k) > maxCsatMetaKeyLength {
			k = k[:maxCsatMetaKeyLength]
		}
		v := string(value)
		if len(v) > maxCsatMetaValLength {
			v = v[:maxCsatMetaValLength]
		}
		meta[k] = v
	})

	metaJSON, err := json.Marshal(meta)
	if err != nil {
		app := r.Context.(*App)
		app.lo.Error("error marshalling CSAT meta", "error", err)
		metaJSON = []byte(`{}`)
	}

	return rating, feedback, metaJSON, ""
}
