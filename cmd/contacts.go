package main

import (
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/abhinavxd/libredesk/internal/user/models"
	realip "github.com/ferluci/fast-realip"
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

// handleCreateContact creates a new contact.
func handleCreateContact(r *fastglue.Request) error {
	var app = r.Context.(*App)

	contact, _, err := contactFromForm(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := app.user.CreateContact(&contact); err != nil {
		return sendErrorEnvelope(r, err)
	}
	created, err := app.user.GetContactOrVisitor(contact.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(created)
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
	contacts, err := app.user.GetContacts(page, pageSize, order, orderBy, filters, app.setting.GetAppTimezone())
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

	contactToUpdate, form, err := contactFromForm(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := app.user.UpdateContact(id, contactToUpdate); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Delete avatar?
	if !contactToUpdate.AvatarURL.Valid && contact.AvatarURL.Valid {
		fileName := filepath.Base(contact.AvatarURL.String)
		app.media.Delete(fileName)
		contact.AvatarURL.Valid = false
		contact.AvatarURL.String = ""
	}

	// Upload avatar?
	if files := form.File["files"]; len(files) > 0 {
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

// handleDeleteContact permanently deletes a contact along with their conversations, messages, and notes.
func handleDeleteContact(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	contact, err := app.user.GetContactOrVisitor(id, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	app.lo.Info("deleting contact", "contact_id", id, "actor_id", auser.ID)

	if err := app.user.DeleteContact(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	if contact.AvatarURL.Valid {
		fileName := filepath.Base(contact.AvatarURL.String)
		if err := app.media.Delete(fileName); err != nil {
			app.lo.Error("error deleting contact avatar", "contact_id", id, "file", fileName, "error", err)
		}
	}

	if err := app.activityLog.ContactDeleted(auser.ID, auser.Email, realip.FromRequest(r.RequestCtx), id, contact.Email.String); err != nil {
		app.lo.Error("error creating contact deleted activity log", "error", err)
	}

	return r.SendEnvelope(true)
}

// handleExportContact sends all stored data for a contact as a JSON file download.
func handleExportContact(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	contact, err := app.user.GetContactOrVisitor(id, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	data, err := app.user.ExportContactData(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := app.activityLog.ContactDataExported(auser.ID, auser.Email, realip.FromRequest(r.RequestCtx), id, contact.Email.String); err != nil {
		app.lo.Error("error creating contact data exported activity log", "error", err)
	}

	filename := fmt.Sprintf("contact-%d-data.json", id)
	r.RequestCtx.Response.Header.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	r.RequestCtx.Response.Header.Set("X-Content-Type-Options", "nosniff")
	r.RequestCtx.SetContentType("application/json; charset=utf-8")
	r.RequestCtx.SetBody(data)
	return nil
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

// contactFromForm parses and validates the multipart contact form shared by create and update.
func contactFromForm(r *fastglue.Request) (models.User, *multipart.Form, error) {
	var app = r.Context.(*App)

	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		app.lo.Error("error parsing form data", "error", err)
		return models.User{}, nil, envelope.NewError(envelope.GeneralError, app.i18n.T("errors.parsingRequest"), nil)
	}

	value := func(key string) string {
		v := ""
		if vals := form.Value[key]; len(vals) > 0 {
			v = strings.TrimSpace(vals[0])
		}
		// Browsers serialise a null field as the string "null".
		if v == "null" {
			return ""
		}
		return v
	}
	optional := func(key string) null.String {
		v := value(key)
		return null.NewString(v, v != "")
	}

	email := value("email")
	if email == "" {
		return models.User{}, nil, envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "email"), nil)
	}
	if !stringutil.ValidEmail(email) {
		return models.User{}, nil, envelope.NewError(envelope.InputError, app.i18n.T("validation.invalidEmail"), nil)
	}
	firstName := value("first_name")
	if firstName == "" {
		return models.User{}, nil, envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "first_name"), nil)
	}

	return models.User{
		FirstName:              firstName,
		LastName:               value("last_name"),
		Email:                  null.StringFrom(email),
		AvatarURL:              optional("avatar_url"),
		PhoneNumber:            optional("phone_number"),
		PhoneNumberCountryCode: optional("phone_number_country_code"),
		Country:                optional("country"),
	}, form, nil
}
