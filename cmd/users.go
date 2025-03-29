package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/image"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	notifier "github.com/abhinavxd/libredesk/internal/notification"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	tmpl "github.com/abhinavxd/libredesk/internal/template"
	"github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

const (
	maxAvatarSizeMB = 20
)

// handleGetUsers returns all users.
func handleGetUsers(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	agents, err := app.user.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(agents)
}

// handleGetUsersCompact returns all users in a compact format.
func handleGetUsersCompact(r *fastglue.Request) error {
	var app = r.Context.(*App)
	agents, err := app.user.GetAllCompact()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(agents)
}

// handleGetUser returns a user.
func handleGetUser(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	user, err := app.user.GetAgent(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(user)
}

// handleUpdateUserAvailability updates the current user availability.
func handleUpdateUserAvailability(r *fastglue.Request) error {
	var (
		app    = r.Context.(*App)
		auser  = r.RequestCtx.UserValue("user").(amodels.User)
		status = string(r.RequestCtx.PostArgs().Peek("status"))
	)
	if err := app.user.UpdateAvailability(auser.ID, status); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleGetCurrentUserTeams returns the teams of a user.
func handleGetCurrentUserTeams(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.GetAgent(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	teams, err := app.team.GetUserTeams(user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(teams)
}

// handleUpdateCurrentUser updates the current user.
func handleUpdateCurrentUser(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.GetAgent(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		app.lo.Error("error parsing form data", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.entities.request}"), nil, envelope.GeneralError)
	}

	files, ok := form.File["files"]

	// Upload avatar?
	if ok && len(files) > 0 {
		fileHeader := files[0]
		file, err := fileHeader.Open()
		if err != nil {
			app.lo.Error("error reading uploaded", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorReading", "name", "{globals.entities.file}"), nil, envelope.GeneralError)
		}
		defer file.Close()

		// Sanitize filename.
		srcFileName := stringutil.SanitizeFilename(fileHeader.Filename)
		srcContentType := fileHeader.Header.Get("Content-Type")
		srcFileSize := fileHeader.Size
		srcExt := strings.TrimPrefix(strings.ToLower(filepath.Ext(srcFileName)), ".")

		if !slices.Contains(image.Exts, srcExt) {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.fileTypeisNotAnImage"), nil, envelope.InputError)
		}

		// Check file size
		if bytesToMegabytes(srcFileSize) > maxAvatarSizeMB {
			app.lo.Error("error uploaded file size is larger than max allowed", "size", bytesToMegabytes(srcFileSize), "max_allowed", maxAvatarSizeMB)
			return r.SendErrorEnvelope(
				http.StatusRequestEntityTooLarge,
				app.i18n.Ts("media.fileSizeTooLarge", "size", fmt.Sprintf("%dMB", maxAvatarSizeMB)),
				nil,
				envelope.GeneralError,
			)
		}

		// Reset ptr.
		file.Seek(0, 0)
		linkedModel := null.StringFrom(mmodels.ModelUser)
		linkedID := null.IntFrom(user.ID)
		disposition := null.NewString("", false)
		contentID := ""
		meta := []byte("{}")
		media, err := app.media.UploadAndInsert(srcFileName, srcContentType, contentID, linkedModel, linkedID, file, int(srcFileSize), disposition, meta)
		if err != nil {
			app.lo.Error("error uploading file", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.errorUploadingFile"), nil, envelope.GeneralError)
		}

		// Delete current avatar.
		if user.AvatarURL.Valid {
			fileName := filepath.Base(user.AvatarURL.String)
			app.media.Delete(fileName)
		}

		// Save file path.
		path, err := stringutil.GetPathFromURL(media.URL)
		if err != nil {
			app.lo.Debug("error getting path from URL", "url", media.URL, "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.errorUploadingFile"), nil, envelope.GeneralError)
		}
		if err := app.user.UpdateAvatar(user.ID, path); err != nil {
			return sendErrorEnvelope(r, err)
		}
	}
	return r.SendEnvelope(true)
}

// handleCreateUser creates a new user.
func handleCreateUser(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		user = models.User{}
	)
	if err := r.Decode(&user, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.entities.request}"), nil, envelope.InputError)
	}

	if user.Email.String == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`email`"), nil, envelope.InputError)
	}

	if user.Roles == nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`role`"), nil, envelope.InputError)
	}

	if user.FirstName == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`first_name`"), nil, envelope.InputError)
	}

	// Right now, only agents can be created.
	if err := app.user.CreateAgent(&user); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Upsert user teams.
	if len(user.Teams) > 0 {
		if err := app.team.UpsertUserTeams(user.ID, user.Teams.Names()); err != nil {
			return sendErrorEnvelope(r, err)
		}
	}

	if user.SendWelcomeEmail {
		// Generate reset token.
		resetToken, err := app.user.SetResetPasswordToken(user.ID)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}

		// Render template and send email.
		content, err := app.tmpl.RenderInMemoryTemplate(tmpl.TmplWelcome, map[string]any{
			"ResetToken": resetToken,
			"Email":      user.Email.String,
		})
		if err != nil {
			app.lo.Error("error rendering template", "error", err)
			return r.SendEnvelope(true)
		}

		if err := app.notifier.Send(notifier.Message{
			RecipientEmails: []string{user.Email.String},
			Subject:         "Welcome",
			Content:         content,
			Provider:        notifier.ProviderEmail,
		}); err != nil {
			app.lo.Error("error sending notification message", "error", err)
			return r.SendEnvelope(true)
		}
	}
	return r.SendEnvelope(true)
}

// handleUpdateUser updates a user.
func handleUpdateUser(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		user = models.User{}
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "{globals.entities.user} `id`"), nil, envelope.InputError)
	}

	if err := r.Decode(&user, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.entities.request}"), nil, envelope.InputError)
	}

	if user.Email.String == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`email`"), nil, envelope.InputError)
	}

	if user.Roles == nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`role`"), nil, envelope.InputError)
	}

	if user.FirstName == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`first_name`"), nil, envelope.InputError)
	}

	// Update user.
	if err = app.user.Update(id, user); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Upsert user teams.
	if err := app.team.UpsertUserTeams(id, user.Teams.Names()); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// handleDeleteUser soft deletes a user.
func handleDeleteUser(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "{globals.entities.user} `id`"), nil, envelope.InputError)
	}

	// Soft delete user.
	if err = app.user.SoftDelete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Unassign all open conversations assigned to the user.
	if err := app.conversation.UnassignOpen(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope("User deleted successfully.")
}

// handleGetCurrentUser returns the current logged in user.
func handleGetCurrentUser(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	u, err := app.user.GetAgent(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(u)
}

// handleDeleteAvatar deletes a user avatar.
func handleDeleteAvatar(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)

	// Get user
	user, err := app.user.GetAgent(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Valid str?
	if user.AvatarURL.String == "" {
		return r.SendEnvelope("Avatar deleted successfully.")
	}

	fileName := filepath.Base(user.AvatarURL.String)

	// Delete file from the store.
	if err := app.media.Delete(fileName); err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err = app.user.UpdateAvatar(user.ID, ""); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleResetPassword generates a reset password token and sends an email to the user.
func handleResetPassword(r *fastglue.Request) error {
	var (
		app       = r.Context.(*App)
		p         = r.RequestCtx.PostArgs()
		auser, ok = r.RequestCtx.UserValue("user").(amodels.User)
		email     = string(p.Peek("email"))
	)
	if ok && auser.ID > 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("user.userAlreadyLoggedIn"), nil, envelope.InputError)
	}

	if email == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`email`"), nil, envelope.InputError)
	}

	user, err := app.user.GetAgentByEmail(email)
	if err != nil {
		// Send 200 even if user not found, to prevent email enumeration.
		return r.SendEnvelope("Reset password email sent successfully.")
	}

	token, err := app.user.SetResetPasswordToken(user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Send email.
	content, err := app.tmpl.RenderInMemoryTemplate(tmpl.TmplResetPassword, map[string]string{
		"ResetToken": token,
	})
	if err != nil {
		app.lo.Error("error rendering template", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.errorSendingPasswordResetEmail"), nil, envelope.GeneralError)
	}

	if err := app.notifier.Send(notifier.Message{
		RecipientEmails: []string{user.Email.String},
		Subject:         "Reset Password",
		Content:         content,
		Provider:        notifier.ProviderEmail,
	}); err != nil {
		app.lo.Error("error sending password reset email", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.errorSendingPasswordResetEmail"), nil, envelope.GeneralError)
	}

	return r.SendEnvelope(true)
}

// handleSetPassword resets the password with the provided token.
func handleSetPassword(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		user, ok = r.RequestCtx.UserValue("user").(amodels.User)
		p        = r.RequestCtx.PostArgs()
		password = string(p.Peek("password"))
		token    = string(p.Peek("token"))
	)

	if ok && user.ID > 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("user.userAlreadyLoggedIn"), nil, envelope.InputError)
	}

	if password == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`password`"), nil, envelope.InputError)
	}

	if err := app.user.ResetPassword(token, password); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}
