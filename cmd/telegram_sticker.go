package main

import (
	"github.com/abhinavxd/libredesk/internal/envelope"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
)

func telegramStickerData(app *App, file *mmodels.Media) ([]byte, error) {
	if file.Size > 65536 {
		return nil, envelope.NewError(envelope.InputError, app.i18n.T("conversation.telegram.stickerUnavailable"), nil)
	}
	data, err := app.media.GetBlob(file.UUID)
	if err != nil {
		return nil, envelope.NewError(envelope.GeneralError, app.i18n.T("conversation.telegram.stickerUnavailable"), nil)
	}
	if len(data) > 65536 {
		return nil, envelope.NewError(envelope.InputError, app.i18n.T("conversation.telegram.stickerUnavailable"), nil)
	}
	return data, nil
}
