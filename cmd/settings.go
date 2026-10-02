package main

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/jakedolan443/fernmail/internal/attachment"
	"github.com/jakedolan443/fernmail/internal/envelope"
	"github.com/jakedolan443/fernmail/internal/httputil"
	"github.com/jakedolan443/fernmail/internal/media"
	mmodels "github.com/jakedolan443/fernmail/internal/media/models"
	"github.com/jakedolan443/fernmail/internal/setting/models"
	"github.com/jakedolan443/fernmail/internal/stringutil"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
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
	var settings map[string]interface{}
	if err := json.Unmarshal(out, &settings); err != nil {
		app.lo.Error("error unmarshalling settings", "err", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
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

	// Trim whitespace from string fields. A blank site name or logo means the Fernmail default.
	req.SiteName = strings.TrimSpace(req.SiteName)
	req.LogoURL = strings.TrimSpace(req.LogoURL)
	if !isValidLogoURL(req.LogoURL) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("admin.general.logoURL.valid"), nil, envelope.InputError)
	}
	req.Timezone = strings.TrimSpace(req.Timezone)
	if req.Timezone != "" && !stringutil.IsValidTimezone(req.Timezone) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid timezone.", nil, envelope.InputError)
	}
	// Trim whitespace and trailing slash from root URL.
	req.RootURL = strings.TrimRight(strings.TrimSpace(req.RootURL), "/")
	if !httputil.IsValidHTTPURL(req.RootURL) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("admin.general.rootURL.valid"), nil, envelope.InputError)
	}

	// Get current language before update.
	app.Lock()
	oldLang := ko.String("app.lang")
	app.Unlock()

	if err := app.setting.Update(req); err != nil {
		return sendErrorEnvelope(r, err)
	}
	// Reload the settings and templates.
	if err := reloadSettings(app); err != nil {
		app.lo.Error("error reloading settings", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
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
		app.lo.Error("error reloading templates", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	return r.SendEnvelope(true)
}

// maxSiteLogoBytes bounds the site logo, which every page loads as its favicon.
const maxSiteLogoBytes = 2 << 20

// siteLogoTypes are the sniffed formats accepted as the site logo. SVG is
// excluded: the logo is public, and an SVG can carry script.
var siteLogoTypes = []string{"image/png", "image/jpeg", "image/webp", "image/x-icon", "image/vnd.microsoft.icon"}

// uploadedLogoRe matches the app-relative URL returned by handleUploadSiteLogo.
var uploadedLogoRe = regexp.MustCompile(`^` + media.PublicURI + `/[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// isValidLogoURL accepts no logo, an uploaded logo, or a logo hosted elsewhere.
func isValidLogoURL(u string) bool {
	return u == "" || uploadedLogoRe.MatchString(u) || httputil.IsValidHTTPURL(u)
}

// siteLogoContentType sniffs the logo's leading bytes rather than trusting the client's content type.
func siteLogoContentType(head []byte) (string, bool) {
	contentType := http.DetectContentType(head)
	return contentType, slices.Contains(siteLogoTypes, contentType)
}

// handleUploadSiteLogo stores a public site logo and returns its app-relative URL.
// The logo takes effect once the general settings are saved with that URL;
// unsaved and replaced logos are swept by the media cleanup.
func handleUploadSiteLogo(r *fastglue.Request) error {
	app := r.Context.(*App)

	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	files := form.File["files"]
	if len(files) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.notFoundFile"), nil, envelope.InputError)
	}
	fileHeader := files[0]
	if fileHeader.Size > maxSiteLogoBytes {
		return r.SendErrorEnvelope(fasthttp.StatusRequestEntityTooLarge, app.i18n.T("admin.general.siteLogo.tooLarge"), nil, envelope.InputError)
	}
	file, err := fileHeader.Open()
	if err != nil {
		app.lo.Error("error reading uploaded logo", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}
	defer file.Close()

	head := make([]byte, 512)
	n, _ := io.ReadFull(file, head)
	contentType, ok := siteLogoContentType(head[:n])
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("admin.general.siteLogo.invalidType"), nil, envelope.InputError)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	logo, err := app.media.UploadAndInsert(stringutil.SanitizeFilename(fileHeader.Filename), contentType, "", null.StringFrom(mmodels.ModelBranding), null.Int{}, file, int(fileHeader.Size), null.StringFrom(attachment.DispositionInline), []byte("{}"), false)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(map[string]string{"url": media.PublicURI + "/" + logo.UUID})
}
