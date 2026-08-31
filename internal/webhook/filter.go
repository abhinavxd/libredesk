package webhook

import (
	"strconv"

	"github.com/abhinavxd/libredesk/internal/webhook/models"
)

type eventScope struct {
	InboxID int
	TeamID  int
	UserID  int
}

func matchesWebhookFilters(w models.Webhook, task DeliveryTask) bool {
	if task.Event == models.EventWebhookTest {
		return true
	}
	scope := payloadScope(task.Payload)
	return matchesIDs(w.InboxIDs, scope.InboxID) &&
		matchesIDs(w.TeamIDs, scope.TeamID) &&
		matchesIDs(w.UserIDs, scope.UserID)
}

func matchesIDs(allowed []int64, id int) bool {
	if len(allowed) == 0 {
		return true
	}
	if id <= 0 {
		return false
	}
	for _, a := range allowed {
		if int(a) == id {
			return true
		}
	}
	return false
}

func payloadScope(payload any) eventScope {
	root := mapFrom(payload)
	if root == nil {
		return eventScope{}
	}
	conv := mapFrom(root["conversation"])
	if conv == nil && looksLikeConversation(root) {
		conv = root
	}
	var convInbox, convTeam, convUser any
	if conv != nil {
		convInbox, convTeam, convUser = conv["inbox_id"], conv["assigned_team_id"], conv["assigned_user_id"]
	}
	return eventScope{
		InboxID: firstID(root["inbox_id"], convInbox),
		TeamID:  firstID(root["assigned_team_id"], convTeam),
		UserID:  firstID(root["assigned_user_id"], convUser, root["assigned_to"]),
	}
}

func firstID(vals ...any) int {
	for _, v := range vals {
		if id := anyToID(v); id > 0 {
			return id
		}
	}
	return 0
}

func anyToID(v any) int {
	s := stringify(v)
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}
