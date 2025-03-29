package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"slices"

	"github.com/abhinavxd/libredesk/internal/attachment"
	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/image"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

const (
	thumbPrefix = "thumb_"
)

// handleMediaUpload handles media uploads.
func handleMediaUpload(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		cleanUp = false
	)

	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		app.lo.Error("error parsing form data.", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.entities.request}"), nil, envelope.GeneralError)
	}

	files, ok := form.File["files"]
	if !ok || len(files) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.notFound", "name", "{globals.entities.file}"), nil, envelope.InputError)
	}

	fileHeader := files[0]
	file, err := fileHeader.Open()
	if err != nil {
		app.lo.Error("error reading uploaded file", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorReading", "name", "{globals.entities.file}"), nil, envelope.GeneralError)
	}
	defer file.Close()

	// Inline?
	var disposition = null.StringFrom(attachment.DispositionAttachment)
	inline, ok := form.Value["inline"]
	if ok && len(inline) > 0 && inline[0] == "true" {
		disposition = null.StringFrom(attachment.DispositionInline)
	}

	// Linked model?
	var linkedModel string
	model, ok := form.Value["linked_model"]
	if ok && len(model) > 0 {
		linkedModel = model[0]
	}

	// Sanitize filename.
	srcFileName := stringutil.SanitizeFilename(fileHeader.Filename)
	srcContentType := fileHeader.Header.Get("Content-Type")
	srcFileSize := fileHeader.Size
	srcExt := strings.TrimPrefix(strings.ToLower(filepath.Ext(srcFileName)), ".")

	// Check file size
	consts := app.consts.Load().(*constants)
	if bytesToMegabytes(srcFileSize) > float64(consts.MaxFileUploadSizeMB) {
		app.lo.Error("error: uploaded file size is larger than max allowed", "size", bytesToMegabytes(srcFileSize), "max_allowed", consts.MaxFileUploadSizeMB)
		return r.SendErrorEnvelope(
			fasthttp.StatusRequestEntityTooLarge,
			app.i18n.Ts("media.fileSizeTooLarge", "size", fmt.Sprint(consts.MaxFileUploadSizeMB), "unit", "MB"),
			nil,
			envelope.GeneralError,
		)
	}

	if !slices.Contains(consts.AllowedUploadFileExtensions, "*") && !slices.Contains(consts.AllowedUploadFileExtensions, srcExt) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("media.fileTypeNotAllowed"), nil, envelope.InputError)
	}

	// Delete files on any error.
	var uuid = uuid.New()
	thumbName := thumbPrefix + uuid.String()
	defer func() {
		if cleanUp {
			app.media.Delete(uuid.String())
			app.media.Delete(thumbName)
		}
	}()

	// Generate and upload thumbnail and store image dimensions in the media meta.
	var meta = []byte("{}")
	if slices.Contains(image.Exts, srcExt) {
		file.Seek(0, 0)
		thumbFile, err := image.CreateThumb(image.DefThumbSize, file)
		if err != nil {
			app.lo.Error("error creating thumb image", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorCreating", "name", "{globals.entities.thumbnail}"), nil, envelope.GeneralError)
		}
		thumbName, err = app.media.Upload(thumbName, srcContentType, thumbFile)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}

		// Store image dimensions in media meta, storing dimensions for image previews in future.
		file.Seek(0, 0)
		width, height, err := image.GetDimensions(file)
		if err != nil {
			cleanUp = true
			app.lo.Error("error getting image dimensions", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorUploading", "name", "{globals.entities.media}"), nil, envelope.GeneralError)
		}
		meta, _ = json.Marshal(map[string]interface{}{
			"width":  width,
			"height": height,
		})
	}

	file.Seek(0, 0)
	_, err = app.media.Upload(uuid.String(), srcContentType, file)
	if err != nil {
		cleanUp = true
		app.lo.Error("error uploading file", "error", err)
		return sendErrorEnvelope(r, err)
	}

	// Insert in DB.
	media, err := app.media.Insert(disposition, srcFileName, srcContentType, "" /**content_id**/, null.NewString(linkedModel, linkedModel != ""), uuid.String(), null.Int{} /**model_id**/, int(srcFileSize), meta)
	if err != nil {
		cleanUp = true
		app.lo.Error("error inserting metadata into database", "error", err)
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(media)
}

// handleServeMedia serves uploaded media.
func handleServeMedia(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
	)

	user, err := app.user.GetAgent(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Fetch media from DB.
	media, err := app.media.Get(0, strings.TrimPrefix(uuid, thumbPrefix))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Check if the user has permission to access the linked model.
	allowed, err := app.authz.EnforceMediaAccess(user, media.Model.String)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// For messages, check access to the conversation this message is part of.
	if media.Model.String == "messages" {
		conversation, err := app.conversation.GetConversationByMessageID(media.ModelID.Int)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		allowed, err = app.authz.EnforceConversationAccess(user, conversation)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
	}

	if !allowed {
		return r.SendErrorEnvelope(http.StatusUnauthorized, app.i18n.T("authz.permissionDenied"), nil, envelope.UnauthorizedError)
	}
	consts := app.consts.Load().(*constants)
	switch consts.UploadProvider {
	case "fs":
		fasthttp.ServeFile(r.RequestCtx, filepath.Join(ko.String("upload.fs.upload_path"), uuid))
	case "s3":
		r.RequestCtx.Redirect(app.media.GetURL(uuid), http.StatusFound)
	}
	return nil
}

// bytesToMegabytes converts bytes to megabytes.
func bytesToMegabytes(bytes int64) float64 {
	return float64(bytes) / 1024 / 1024
}
