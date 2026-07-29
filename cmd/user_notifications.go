package main

import (
	"encoding/json"
	"strconv"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	nmodels "github.com/abhinavxd/libredesk/internal/notification/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func handleGetUserNotifications(r *fastglue.Request) error {
	var (
		app    = r.Context.(*App)
		auser  = r.RequestCtx.UserValue("user").(amodels.User)
		limit  = 20
		offset = 0
	)
	if l := r.RequestCtx.QueryArgs().GetUintOrZero("limit"); l > 0 && l <= 100 {
		limit = l
	}
	if o := r.RequestCtx.QueryArgs().GetUintOrZero("offset"); o > 0 {
		offset = o
	}
	notifications, err := app.userNotification.GetAll(auser.ID, limit, offset)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(notifications)
}

func handleGetUserNotificationStats(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	stats, err := app.userNotification.GetStats(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(stats)
}

func handleMarkNotificationAsRead(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	if err := app.userNotification.MarkAsRead(id, auser.ID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

func handleMarkAllNotificationsAsRead(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)

	if err := app.userNotification.MarkAllAsRead(auser.ID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

func handleDeleteNotification(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	if err := app.userNotification.Delete(id, auser.ID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

func handleDeleteAllNotifications(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)

	if err := app.userNotification.DeleteAll(auser.ID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

func handleGetNotificationPreferences(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	prefs, err := app.notificationPref.GetMatrix(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(map[string]any{
		"preferences":   prefs,
		"email_enabled": ko.Bool("notification.email.enabled"),
	})
}

func handleUpdateNotificationPreferences(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		prefs []nmodels.NotificationPreference
	)
	if err := json.Unmarshal(r.RequestCtx.PostBody(), &prefs); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	if err := app.notificationPref.Update(auser.ID, prefs); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
