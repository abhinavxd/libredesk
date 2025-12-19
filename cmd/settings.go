package main

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"time"
	"encoding/json"
	"net/mail"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/setting/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetGeneralSettings fetches general settings, this endpoint is not behind auth as it has no sensitive data and is required for the app to function.
func handleGetGeneralSettings(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	out, err := app.setting.GetByPrefix("app")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	// Unmarshal to set the app.update to the settings, so the frontend can show that an update is available.
	var settings map[string]interface{}
	if err := json.Unmarshal(out, &settings); err != nil {
		app.lo.Error("error unmarshalling settings", "err", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorFetching", "name", app.i18n.T("globals.terms.setting")), nil))
	}
	// Set the app.update to the settings, adding `app` prefix to the key to match the settings structure in db.
	settings["app.update"] = app.update
	// Set app version.
	settings["app.version"] = versionString
	// Set restart required flag.
	settings["app.restart_required"] = app.restartRequired
	return r.SendEnvelope(settings)
}

// handleUpdateGeneralSettings updates general settings.
func handleUpdateGeneralSettings(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = models.General{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	// Get current language before update.
	app.Lock()
	oldLang := ko.String("app.lang")
	app.Unlock()

	// Remove any trailing slash `/` from the root url.
	req.RootURL = strings.TrimRight(req.RootURL, "/")

	if err := app.setting.Update(req); err != nil {
		return sendErrorEnvelope(r, err)
	}
	// Reload the settings and templates.
	if err := reloadSettings(app); err != nil {
		return envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.couldNotReload", "name", app.i18n.T("globals.terms.setting")), nil)
	}

	// Check if language changed and reload i18n if needed.
	app.Lock()
	newLang := ko.String("app.lang")
	if oldLang != newLang {
		app.lo.Info("language changed, reloading i18n", "old_lang", oldLang, "new_lang", newLang)
		app.i18n = initI18n(app.fs)
		app.lo.Info("reloaded i18n", "old_lang", oldLang, "new_lang", newLang)
	}
	app.Unlock()

	if err := reloadTemplates(app); err != nil {
		return envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.couldNotReload", "name", app.i18n.T("globals.terms.setting")), nil)
	}
	return r.SendEnvelope(true)
}

// handleGetEmailNotificationSettings fetches email notification settings.
func handleGetEmailNotificationSettings(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		notif = models.EmailNotification{}
	)

	out, err := app.setting.GetByPrefix("notification.email")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Unmarshal and filter out password.
	if err := json.Unmarshal(out, &notif); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorFetching", "name", app.i18n.T("globals.terms.setting")), nil))
	}
	if notif.Password != "" {
		notif.Password = strings.Repeat(stringutil.PasswordDummy, 10)
	}
	return r.SendEnvelope(notif)
}

// handleUpdateEmailNotificationSettings updates email notification settings.
func handleUpdateEmailNotificationSettings(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = models.EmailNotification{}
		cur = models.EmailNotification{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	out, err := app.setting.GetByPrefix("notification.email")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := json.Unmarshal(out, &cur); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorUpdating", "name", app.i18n.T("globals.terms.setting")), nil))
	}

	// Make sure it's a valid from email address.
	if _, err := mail.ParseAddress(req.EmailAddress); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.invalidFromAddress"), nil, envelope.InputError)
	}

	// If empty then retain previous password.
	if req.Password == "" {
		req.Password = cur.Password
	}

	if err := app.setting.Update(req); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Email notification settings require app restart to take effect.
	app.Lock()
	app.restartRequired = true
	app.Unlock()

	return r.SendEnvelope(true)
}


// TestEmailRequest represents the request body for testing email settings.
type TestEmailRequest struct {
	models.EmailNotification
	TestEmail string `json:"test_email"`
}

// TestEmailResponse represents the response for testing email settings.
type TestEmailResponse struct {
	Success bool     `json:"success"`
	Logs    []string `json:"logs"`
}

// handleTestEmailNotificationSettings tests the email notification settings by sending a test email.
func handleTestEmailNotificationSettings(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		req  = TestEmailRequest{}
		cur  = models.EmailNotification{}
		logs = []string{}
	)

	addLog := func(msg string) {
		logs = append(logs, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg))
	}

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	// Validate test email
	if req.TestEmail == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Test email address is required", nil, envelope.InputError)
	}
	if _, err := mail.ParseAddress(req.TestEmail); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid test email address", nil, envelope.InputError)
	}

	addLog(fmt.Sprintf("Starting SMTP test to %s", req.TestEmail))

	// Get current settings to fill in password if not provided
	out, err := app.setting.GetByPrefix("notification.email")
	if err != nil {
		addLog(fmt.Sprintf("Error fetching current settings: %v", err))
		return r.SendEnvelope(TestEmailResponse{Success: false, Logs: logs})
	}
	if err := json.Unmarshal(out, &cur); err != nil {
		addLog(fmt.Sprintf("Error parsing current settings: %v", err))
		return r.SendEnvelope(TestEmailResponse{Success: false, Logs: logs})
	}

	// Use current password if not provided in request
	password := req.Password
	if password == "" || strings.Contains(password, stringutil.PasswordDummy) {
		password = cur.Password
	}

	// Build server address
	serverAddr := fmt.Sprintf("%s:%d", req.Host, req.Port)
	addLog(fmt.Sprintf("Connecting to SMTP server: %s", serverAddr))

	// Create TLS config
	tlsConfig := &tls.Config{
		ServerName:         req.Host,
		InsecureSkipVerify: req.TLSSkipVerify,
	}

	var client *smtp.Client

	// Connect based on TLS type
	switch req.TLSType {
	case "tls":
		addLog("Using SSL/TLS connection")
		conn, err := tls.Dial("tcp", serverAddr, tlsConfig)
		if err != nil {
			addLog(fmt.Sprintf("TLS connection failed: %v", err))
			return r.SendEnvelope(TestEmailResponse{Success: false, Logs: logs})
		}
		defer conn.Close()
		client, err = smtp.NewClient(conn, req.Host)
		if err != nil {
			addLog(fmt.Sprintf("Failed to create SMTP client: %v", err))
			return r.SendEnvelope(TestEmailResponse{Success: false, Logs: logs})
		}
	default:
		addLog("Using plain connection")
		var err error
		client, err = smtp.Dial(serverAddr)
		if err != nil {
			addLog(fmt.Sprintf("Connection failed: %v", err))
			return r.SendEnvelope(TestEmailResponse{Success: false, Logs: logs})
		}
	}
	defer client.Close()

	// Send HELO/EHLO
	hostname := req.HelloHostname
	if hostname == "" {
		hostname = "localhost"
	}
	addLog(fmt.Sprintf("Sending EHLO %s", hostname))
	if err := client.Hello(hostname); err != nil {
		addLog(fmt.Sprintf("EHLO failed: %v", err))
		return r.SendEnvelope(TestEmailResponse{Success: false, Logs: logs})
	}

	// STARTTLS if required
	if req.TLSType == "starttls" {
		addLog("Starting TLS (STARTTLS)")
		if err := client.StartTLS(tlsConfig); err != nil {
			addLog(fmt.Sprintf("STARTTLS failed: %v", err))
			return r.SendEnvelope(TestEmailResponse{Success: false, Logs: logs})
		}
		addLog("TLS connection established")
	}

	// Authenticate if credentials provided
	if req.Username != "" && password != "" {
		addLog(fmt.Sprintf("Authenticating as %s using %s", req.Username, req.AuthProtocol))
		var auth smtp.Auth
		switch req.AuthProtocol {
		case "plain":
			auth = smtp.PlainAuth("", req.Username, password, req.Host)
		case "login":
			auth = &loginAuth{username: req.Username, password: password}
		case "cram":
			auth = smtp.CRAMMD5Auth(req.Username, password)
		case "none":
			addLog("No authentication required")
		default:
			auth = smtp.PlainAuth("", req.Username, password, req.Host)
		}
		if auth != nil {
			if err := client.Auth(auth); err != nil {
				addLog(fmt.Sprintf("Authentication failed: %v", err))
				return r.SendEnvelope(TestEmailResponse{Success: false, Logs: logs})
			}
			addLog("Authentication successful")
		}
	}

	// Set sender
	fromAddr := req.EmailAddress
	if fromAddr == "" {
		fromAddr = req.Username
	}
	addLog(fmt.Sprintf("Setting sender: %s", fromAddr))
	if err := client.Mail(fromAddr); err != nil {
		addLog(fmt.Sprintf("MAIL FROM failed: %v", err))
		return r.SendEnvelope(TestEmailResponse{Success: false, Logs: logs})
	}

	// Set recipient
	addLog(fmt.Sprintf("Setting recipient: %s", req.TestEmail))
	if err := client.Rcpt(req.TestEmail); err != nil {
		addLog(fmt.Sprintf("RCPT TO failed: %v", err))
		return r.SendEnvelope(TestEmailResponse{Success: false, Logs: logs})
	}

	// Send test message
	addLog("Sending test message")
	w, err := client.Data()
	if err != nil {
		addLog(fmt.Sprintf("DATA command failed: %v", err))
		return r.SendEnvelope(TestEmailResponse{Success: false, Logs: logs})
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: LibreDesk SMTP Test\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nThis is a test email from LibreDesk to verify your SMTP notification settings are working correctly.\r\n\r\nSent at: %s",
		fromAddr, req.TestEmail, time.Now().Format(time.RFC1123))

	if _, err := w.Write([]byte(msg)); err != nil {
		addLog(fmt.Sprintf("Failed to write message: %v", err))
		return r.SendEnvelope(TestEmailResponse{Success: false, Logs: logs})
	}
	if err := w.Close(); err != nil {
		addLog(fmt.Sprintf("Failed to close message: %v", err))
		return r.SendEnvelope(TestEmailResponse{Success: false, Logs: logs})
	}

	addLog("Test email sent successfully!")
	client.Quit()

	return r.SendEnvelope(TestEmailResponse{Success: true, Logs: logs})
}

// loginAuth implements smtp.Auth for LOGIN authentication
type loginAuth struct {
	username, password string
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte{}, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		}
	}
	return nil, nil
}
// handleGetAISettings fetches AI settings.
func handleGetAISettings(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)

	out, err := app.setting.GetByPrefix("ai")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Unmarshal and mask API keys
	var settings map[string]interface{}
	if err := json.Unmarshal(out, &settings); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorFetching", "name", app.i18n.T("globals.terms.setting")), nil))
	}

	// Mask API keys
	if key, ok := settings["ai.openai_api_key"].(string); ok && key != "" {
		settings["ai.openai_api_key"] = strings.Repeat(stringutil.PasswordDummy, 10)
	}
	if key, ok := settings["ai.openrouter_api_key"].(string); ok && key != "" {
		settings["ai.openrouter_api_key"] = strings.Repeat(stringutil.PasswordDummy, 10)
	}

	return r.SendEnvelope(settings)
}

// handleUpdateAISettings updates AI settings.
func handleUpdateAISettings(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = models.AISettings{}
		cur = models.AISettings{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	// Get current settings to preserve passwords if not provided
	curJSON, err := app.setting.GetByPrefix("ai")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := json.Unmarshal(curJSON, &cur); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorUpdating", "name", app.i18n.T("globals.terms.setting")), nil))
	}

	// If empty or dummy, retain previous API keys
	if req.OpenAIAPIKey == "" || strings.HasPrefix(req.OpenAIAPIKey, stringutil.PasswordDummy) {
		req.OpenAIAPIKey = cur.OpenAIAPIKey
	}
	if req.OpenRouterAPIKey == "" || strings.HasPrefix(req.OpenRouterAPIKey, stringutil.PasswordDummy) {
		req.OpenRouterAPIKey = cur.OpenRouterAPIKey
	}

	if err := app.setting.Update(req); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Reload the settings
	if err := reloadSettings(app); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.couldNotReload", "name", app.i18n.T("globals.terms.setting")), nil))
	}

	return r.SendEnvelope(true)
}
