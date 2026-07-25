package main

import (
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
)

func markAssignmentNotificationRead(app *App, conv *cmodels.Conversation, user umodels.User) {
	if conv == nil || conv.ID == 0 || conv.AssignedUserID.Int != user.ID {
		return
	}
	_ = app.userNotification.MarkAssignmentAsRead(conv.ID, user.ID)
}
