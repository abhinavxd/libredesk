package main

import (
	"fmt"
	"mime/multipart"
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
	realip "github.com/ferluci/fast-realip"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

const (
	maxAvatarSizeMB = 2
)

type updateAvailabilityRequest struct {
	Status string `json:"status"`
}

type resetPasswordRequest struct {
	Email string `json:"email"`
}

type setPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

type availabilityRequest struct {
	Status string `json:"status"`
}

type agentReq struct {
	FirstName          string   `json:"first_name"`
	LastName           string   `json:"last_name"`
	Email              string   `json:"email"`
	SendWelcomeEmail   bool     `json:"send_welcome_email"`
	Teams              []string `json:"teams"`
	Roles              []string `json:"roles"`
	Enabled            bool     `json:"enabled"`
	AvailabilityStatus string   `json:"availability_status"`
	NewPassword        string   `json:"new_password,omitempty"`
}

// handleGetAgents returns all agents.
func handleGetAgents(r *fastglue.Request) error {
	var app = r.Context.(*App)
	agents, err := app.user.GetAgents()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(agents)
}

// handleGetAgentsCompact returns all agents in a compact format.
func handleGetAgentsCompact(r *fastglue.Request) error {
	var app = r.Context.(*App)
	agents, err := app.user.GetAgentsCompact()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(agents)
}

// handleGetAgent returns an agent.
func handleGetAgent(r *fastglue.Request) error {
	var app = r.Context.(*App)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	agent, err := app.user.GetAgent(id, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(agent)
}

// handleUpdateAgentAvailability updates the current agent availability.
func handleUpdateAgentAvailability(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		auser    = r.RequestCtx.UserValue("user").(amodels.User)
		ip       = realip.FromRequest(r.RequestCtx)
		availReq availabilityRequest
	)

	// Decode JSON request
	if err := r.Decode(&availReq, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	// Fetch entire agent
	agent, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Same status?
	if agent.AvailabilityStatus == availReq.Status {
		return r.SendEnvelope(agent)
	}

	// Update availability status
	if err := app.user.UpdateAvailability(auser.ID, availReq.Status); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Skip activity log if agent returns online from away (to avoid spam).
	if !(agent.AvailabilityStatus == models.Away && availReq.Status == models.Online) {
		if err := app.activityLog.UserAvailability(auser.ID, auser.Email, availReq.Status, ip, "", 0); err != nil {
			app.lo.Error("error creating activity log", "error", err)
		}
	}

	// Fetch updated agent and return
	agent, err = app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(agent)
}

// handleGetCurrentAgentTeams returns the teams of current agent.
func handleGetCurrentAgentTeams(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	teams, err := app.team.GetUserTeams(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(teams)
}

// handleUpdateCurrentAgent updates the current agent.
func handleUpdateCurrentAgent(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		app.lo.Error("error parsing form data", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.GeneralError)
	}

	files, ok := form.File["files"]

	// Upload avatar?
	if ok && len(files) > 0 {
		agent, err := app.user.GetAgent(auser.ID, "")
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		if err := uploadUserAvatar(r, &agent, files); err != nil {
			return sendErrorEnvelope(r, err)
		}
	}

	// Fetch updated agent and return.
	agent, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(agent)
}

// handleCreateAgent creates a new agent.
func handleCreateAgent(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = agentReq{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	// Validate agent request
	if err := validateAgentRequest(r, &req); err != nil {
		return err
	}

	agent, err := app.user.CreateAgent(req.FirstName, req.LastName, req.Email, req.Roles)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Upsert user teams.
	if len(req.Teams) > 0 {
		app.team.UpsertUserTeams(agent.ID, req.Teams)
	}

	if req.SendWelcomeEmail {
		// Generate reset token.
		resetToken, err := app.user.SetResetPasswordToken(agent.ID)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}

		// Render template and send email.
		content, err := app.tmpl.RenderInMemoryTemplate(tmpl.TmplWelcome, map[string]any{
			"ResetToken": resetToken,
			"Email":      req.Email,
		})
		if err != nil {
			app.lo.Error("error rendering template", "error", err)
		}

		if err := app.notifier.Send(notifier.Message{
			RecipientEmails: []string{req.Email},
			Subject:         app.i18n.T("globals.messages.welcomeToLibredesk"),
			Content:         content,
			Provider:        notifier.ProviderEmail,
		}); err != nil {
			app.lo.Error("error sending notification message", "error", err)
		}
	}

	// Refetch agent as other details might've changed.
	agent, err = app.user.GetAgent(agent.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(agent)
}

// handleUpdateAgent updates an agent.
func handleUpdateAgent(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		req   = agentReq{}
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		ip    = realip.FromRequest(r.RequestCtx)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`id`"), nil, envelope.InputError)
	}

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	// Validate agent request
	if err := validateAgentRequest(r, &req); err != nil {
		return err
	}

	agent, err := app.user.GetAgent(id, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	oldAvailabilityStatus := agent.AvailabilityStatus

	// Update agent with individual fields
	if err = app.user.UpdateAgent(id, req.FirstName, req.LastName, req.Email, req.Roles, req.Enabled, req.AvailabilityStatus, req.NewPassword); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Invalidate authz cache.
	defer app.authz.InvalidateUserCache(id)

	// Create activity log if user availability status changed.
	if oldAvailabilityStatus != req.AvailabilityStatus {
		if err := app.activityLog.UserAvailability(auser.ID, auser.Email, req.AvailabilityStatus, ip, req.Email, id); err != nil {
			app.lo.Error("error creating activity log", "error", err)
		}
	}

	// Upsert agent teams.
	if err := app.team.UpsertUserTeams(id, req.Teams); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Refetch agent and return.
	agent, err = app.user.GetAgent(id, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(agent)
}

// handleDeleteAgent soft deletes an agent.
func handleDeleteAgent(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		auser   = r.RequestCtx.UserValue("user").(amodels.User)
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "{globals.terms.user} `id`"), nil, envelope.InputError)
	}

	// Disallow if self-deleting.
	if id == auser.ID {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("user.userCannotDeleteSelf"), nil, envelope.InputError)
	}

	// Soft delete user.
	if err = app.user.SoftDeleteAgent(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Unassign all open conversations assigned to the user.
	if err := app.conversation.UnassignOpen(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// handleGetCurrentAgent returns the current logged in agent.
func handleGetCurrentAgent(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	u, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(u)
}

// handleDeleteCurrentAgentAvatar deletes the current agent's avatar.
func handleDeleteCurrentAgentAvatar(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)

	// Get user
	agent, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Valid str?
	if agent.AvatarURL.String == "" {
		return r.SendEnvelope(true)
	}

	fileName := filepath.Base(agent.AvatarURL.String)

	// Delete file from the store.
	if err := app.media.Delete(fileName); err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err = app.user.UpdateAvatar(agent.ID, ""); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleResetPassword generates a reset password token and sends an email to the agent.
func handleResetPassword(r *fastglue.Request) error {
	var (
		app       = r.Context.(*App)
		auser, ok = r.RequestCtx.UserValue("user").(amodels.User)
		resetReq  resetPasswordRequest
	)
	if ok && auser.ID > 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("user.userAlreadyLoggedIn"), nil, envelope.InputError)
	}

	// Decode JSON request
	if err := r.Decode(&resetReq, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	if resetReq.Email == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`email`"), nil, envelope.InputError)
	}

	agent, err := app.user.GetAgent(0, resetReq.Email)
	if err != nil {
		// Send 200 even if user not found, to prevent email enumeration.
		return r.SendEnvelope(true)
	}

	token, err := app.user.SetResetPasswordToken(agent.ID)
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
		RecipientEmails: []string{agent.Email.String},
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
		app       = r.Context.(*App)
		agent, ok = r.RequestCtx.UserValue("user").(amodels.User)
		req       setPasswordRequest
	)

	if ok && agent.ID > 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("user.userAlreadyLoggedIn"), nil, envelope.InputError)
	}

	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil))
	}

	if req.Password == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "{globals.terms.password}"), nil, envelope.InputError)
	}

	if err := app.user.ResetPassword(req.Token, req.Password); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// uploadUserAvatar uploads the user avatar.
func uploadUserAvatar(r *fastglue.Request, user *models.User, files []*multipart.FileHeader) error {
	var app = r.Context.(*App)

	fileHeader := files[0]
	file, err := fileHeader.Open()
	if err != nil {
		app.lo.Error("error opening uploaded file", "error", err)
		return envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorReading", "name", "{globals.terms.file}"), nil)
	}
	defer file.Close()

	// Sanitize filename.
	srcFileName := stringutil.SanitizeFilename(fileHeader.Filename)
	srcContentType := fileHeader.Header.Get("Content-Type")
	srcFileSize := fileHeader.Size
	srcExt := strings.TrimPrefix(strings.ToLower(filepath.Ext(srcFileName)), ".")

	if !slices.Contains(image.Exts, srcExt) {
		return envelope.NewError(envelope.InputError, app.i18n.T("globals.messages.fileTypeisNotAnImage"), nil)
	}

	// Check file size
	if bytesToMegabytes(srcFileSize) > maxAvatarSizeMB {
		app.lo.Error("error uploaded file size is larger than max allowed", "size", bytesToMegabytes(srcFileSize), "max_allowed", maxAvatarSizeMB)
		return envelope.NewError(
			envelope.InputError,
			app.i18n.Ts("media.fileSizeTooLarge", "size", fmt.Sprintf("%dMB", maxAvatarSizeMB)),
			nil,
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
		return envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorUploading", "name", "{globals.terms.file}"), nil)
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
		return envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorUploading", "name", "{globals.terms.file}"), nil)
	}

	if err := app.user.UpdateAvatar(user.ID, path); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return nil
}

// handleGenerateAPIKey generates a new API key for a user
func handleGenerateAPIKey(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	// Check if user exists
	user, err := app.user.GetAgent(id, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Generate API key and secret
	apiKey, apiSecret, err := app.user.GenerateAPIKey(user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Return the API key and secret (only shown once)
	response := struct {
		APIKey    string `json:"api_key"`
		APISecret string `json:"api_secret"`
	}{
		APIKey:    apiKey,
		APISecret: apiSecret,
	}

	return r.SendEnvelope(response)
}

// handleRevokeAPIKey revokes a user's API key
func handleRevokeAPIKey(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	// Check if user exists
	_, err := app.user.GetAgent(id, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Revoke API key
	if err := app.user.RevokeAPIKey(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// validateAgentRequest validates common agent request fields and normalizes the email
func validateAgentRequest(r *fastglue.Request, req *agentReq) error {
	var app = r.Context.(*App)

	// Normalize email
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`email`"), nil, envelope.InputError)
	}

	if !stringutil.ValidEmail(req.Email) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`email`"), nil, envelope.InputError)
	}

	if req.Roles == nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`role`"), nil, envelope.InputError)
	}

	if req.FirstName == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`first_name`"), nil, envelope.InputError)
	}

	return nil
}
