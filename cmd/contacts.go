package main

import (
	"path/filepath"
	"strconv"
	"strings"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

type createContactNoteReq struct {
	Note string `json:"note"`
}

type blockContactReq struct {
	Enabled bool `json:"enabled"`
}

type contactFormValues struct {
	FirstName              string
	LastName               string
	Email                  string
	PhoneNumber            string
	PhoneNumberCountryCode string
	Country                string
	AvatarURL              string
}

func parseContactFormValues(values map[string][]string) contactFormValues {
	return contactFormValues{
		FirstName:              getContactFormValue(values, "first_name"),
		LastName:               getContactFormValue(values, "last_name"),
		Email:                  strings.TrimSpace(getContactFormValue(values, "email")),
		PhoneNumber:            nullifyContactFormValue(getContactFormValue(values, "phone_number")),
		PhoneNumberCountryCode: nullifyContactFormValue(getContactFormValue(values, "phone_number_country_code")),
		Country:                nullifyContactFormValue(getContactFormValue(values, "country")),
		AvatarURL:              nullifyContactFormValue(getContactFormValue(values, "avatar_url")),
	}
}

func getContactFormValue(values map[string][]string, key string) string {
	if v, ok := values[key]; ok && len(v) > 0 {
		return v[0]
	}
	return ""
}

func nullifyContactFormValue(value string) string {
	if value == "null" {
		return ""
	}
	return value
}

// handleCreateContact creates a new contact.
func handleCreateContact(r *fastglue.Request) error {
	var app = r.Context.(*App)

	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		app.lo.Error("error parsing form data", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("errors.parsingRequest"), nil, envelope.GeneralError)
	}

	fields := parseContactFormValues(form.Value)

	// Validate email format if provided.
	if fields.Email != "" && !stringutil.ValidEmail(fields.Email) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.invalidEmail"), nil, envelope.InputError)
	}

	contact := models.User{
		FirstName:              fields.FirstName,
		LastName:               fields.LastName,
		Email:                  null.NewString(fields.Email, fields.Email != ""),
		PhoneNumber:            null.NewString(fields.PhoneNumber, fields.PhoneNumber != ""),
		PhoneNumberCountryCode: null.NewString(fields.PhoneNumberCountryCode, fields.PhoneNumberCountryCode != ""),
		Country:                null.NewString(fields.Country, fields.Country != ""),
	}

	if err := app.user.CreateManualContact(&contact); err != nil {
		return sendErrorEnvelope(r, err)
	}

	createdContact, err := app.user.GetContactOrVisitor(contact.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Upload avatar if provided. Treat as best-effort: a failure does not
	// roll back the already-created contact.
	files, ok := form.File["files"]
	if ok && len(files) > 0 {
		if err := uploadUserAvatar(r, createdContact, files); err != nil {
			app.lo.Error("failed to upload avatar for new contact", "contact_id", contact.ID, "error", err)
		} else {
			createdContact, err = app.user.GetContactOrVisitor(contact.ID, "")
			if err != nil {
				return sendErrorEnvelope(r, err)
			}
		}
	}

	return r.SendEnvelope(createdContact)
}

// handleGetContacts returns a list of contacts from the database.
func handleGetContacts(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		order   = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		filters = string(r.RequestCtx.QueryArgs().Peek("filters"))
		total   = 0
	)
	page, pageSize := getPagination(r)
	contacts, err := app.user.GetContacts(page, pageSize, order, orderBy, filters)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if len(contacts) > 0 {
		total = contacts[0].Total
	}
	return r.SendEnvelope(envelope.PageResults{
		Results:    contacts,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}

// handleGetTags returns a contact from the database.
func handleGetContact(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	c, err := app.user.GetContactOrVisitor(id, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(c)
}

// handleUpdateContact updates a contact in the database.
func handleUpdateContact(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	contact, err := app.user.GetContactOrVisitor(id, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		app.lo.Error("error parsing form data", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("errors.parsingRequest"), nil, envelope.GeneralError)
	}

	fields := parseContactFormValues(form.Value)

	// Validate mandatory fields.
	if fields.Email == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "email"), nil, envelope.InputError)
	}
	if !stringutil.ValidEmail(fields.Email) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.invalidEmail"), nil, envelope.InputError)
	}
	if fields.FirstName == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "first_name"), nil, envelope.InputError)
	}

	contactToUpdate := models.User{
		FirstName:              fields.FirstName,
		LastName:               fields.LastName,
		Email:                  null.StringFrom(fields.Email),
		AvatarURL:              null.NewString(fields.AvatarURL, fields.AvatarURL != ""),
		PhoneNumber:            null.NewString(fields.PhoneNumber, fields.PhoneNumber != ""),
		PhoneNumberCountryCode: null.NewString(fields.PhoneNumberCountryCode, fields.PhoneNumberCountryCode != ""),
		Country:                null.NewString(fields.Country, fields.Country != ""),
	}

	if err := app.user.UpdateContact(id, contactToUpdate); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Delete avatar?
	if fields.AvatarURL == "" && contact.AvatarURL.Valid {
		fileName := filepath.Base(contact.AvatarURL.String)
		app.media.Delete(fileName)
		contact.AvatarURL.Valid = false
		contact.AvatarURL.String = ""
	}

	// Upload avatar?
	files, ok := form.File["files"]
	if ok && len(files) > 0 {
		if err := uploadUserAvatar(r, contact, files); err != nil {
			return sendErrorEnvelope(r, err)
		}
	}

	// Refetch contact and return it
	contact, err = app.user.GetContactOrVisitor(id, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(contact)
}

// handleGetContactNotes returns all notes for a contact.
func handleGetContactNotes(r *fastglue.Request) error {
	var (
		app          = r.Context.(*App)
		contactID, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if contactID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	notes, err := app.user.GetNotes(contactID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(notes)
}

// handleCreateContactNote creates a note for a contact.
func handleCreateContactNote(r *fastglue.Request) error {
	var (
		app          = r.Context.(*App)
		contactID, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		auser        = r.RequestCtx.UserValue("user").(amodels.User)
		req          = createContactNoteReq{}
	)
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	if len(req.Note) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "note"), nil, envelope.InputError)
	}
	n, err := app.user.CreateNote(contactID, auser.ID, req.Note)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	n, err = app.user.GetNote(n.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(n)
}

// handleDeleteContactNote deletes a note for a contact.
func handleDeleteContactNote(r *fastglue.Request) error {
	var (
		app          = r.Context.(*App)
		contactID, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		noteID, _    = strconv.Atoi(r.RequestCtx.UserValue("note_id").(string))
		auser        = r.RequestCtx.UserValue("user").(amodels.User)
	)
	if contactID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	if noteID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	agent, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Allow deletion of only own notes and not those created by others, but also allow `Admin` to delete any note.
	if !agent.HasAdminRole() {
		note, err := app.user.GetNote(noteID)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		if note.UserID != auser.ID {
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("errors.canOnlyDeleteOwnNote"), nil, envelope.InputError)
		}
	}

	app.lo.Info("deleting contact note", "note_id", noteID, "contact_id", contactID, "actor_id", auser.ID)

	if err := app.user.DeleteNote(noteID, contactID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleBlockContact blocks a contact.
func handleBlockContact(r *fastglue.Request) error {
	var (
		app          = r.Context.(*App)
		contactID, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		auser        = r.RequestCtx.UserValue("user").(amodels.User)
		req          = blockContactReq{}
	)

	if contactID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}

	app.lo.Info("setting contact block status", "contact_id", contactID, "enabled", req.Enabled, "actor_id", auser.ID)

	contact, err := app.user.GetContactOrVisitor(contactID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := app.user.ToggleEnabled(contactID, contact.Type, req.Enabled); err != nil {
		return sendErrorEnvelope(r, err)
	}

	contact, err = app.user.GetContactOrVisitor(contactID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(contact)
}
