package conversation

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"time"

	amodels "github.com/abhinavxd/libredesk/internal/automation/models"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	custommodels "github.com/abhinavxd/libredesk/internal/custom_attribute/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
)

type customAttributeStore interface {
	Get(id int) (custommodels.CustomAttribute, error)
}

func (m *Manager) SetCustomAttributeStore(store customAttributeStore) {
	m.customAttributeStore = store
}

func (m *Manager) applyCustomAttributeAction(action amodels.RuleAction, conv models.Conversation, actor umodels.User) error {
	if len(action.Value) == 0 || m.customAttributeStore == nil {
		return fmt.Errorf("custom attribute action requires an attribute")
	}
	id, err := strconv.Atoi(action.Value[0])
	if err != nil || id <= 0 {
		return fmt.Errorf("invalid custom attribute ID %q", action.Value[0])
	}
	attribute, err := m.customAttributeStore.Get(id)
	if err != nil {
		return err
	}
	contact := action.Type == amodels.ActionSetContactAttribute || action.Type == amodels.ActionRemoveContactAttribute
	remove := action.Type == amodels.ActionRemoveContactAttribute || action.Type == amodels.ActionRemoveConversationAttribute
	appliesTo := "conversation"
	if contact {
		appliesTo = "contact"
	}
	if attribute.AppliesTo != appliesTo {
		return fmt.Errorf("custom attribute %d does not belong to %s", id, appliesTo)
	}
	var value any
	if !remove {
		if len(action.Value) != 2 {
			return fmt.Errorf("custom attribute action requires a value")
		}
		value, err = parseAutomationAttributeValue(attribute, action.Value[1])
		if err != nil {
			return err
		}
	}
	if contact {
		if remove {
			err = m.userStore.RemoveCustomAttribute(conv.ContactID, attribute.Key)
		} else {
			err = m.userStore.SaveCustomAttributes(conv.ContactID, map[string]any{attribute.Key: value}, false /** replace **/)
		}
		if err != nil {
			return err
		}
		user, err := m.userStore.Get(conv.ContactID, "" /** email **/, []string{} /** userType **/)
		if err != nil {
			return err
		}
		m.BroadcastContactUpdate(conv.ContactID, map[string]any{"custom_attributes": user.CustomAttributes})
		m.automation.EvaluateConversationUpdateRulesByID(conv.ID, conv.UUID, amodels.EventConversationContactAttributesChange, actor)
		return nil
	}
	var attributes jsontext.Value
	if remove {
		err = m.q.RemoveConversationCustomAttribute.Get(&attributes, conv.UUID, attribute.Key)
	} else {
		encoded, encodeErr := json.Marshal(map[string]any{attribute.Key: value})
		if encodeErr != nil {
			return encodeErr
		}
		err = m.q.UpsertConversationCustomAttribute.Get(&attributes, conv.UUID, encoded)
	}
	if err != nil {
		return fmt.Errorf("updating conversation custom attribute: %w", err)
	}
	m.BroadcastConversationUpdate(conv.UUID, map[string]any{"custom_attributes": attributes})
	m.automation.EvaluateConversationUpdateRulesByID(conv.ID, conv.UUID, amodels.EventConversationAttributesChange, actor)
	return nil
}

func parseAutomationAttributeValue(attribute custommodels.CustomAttribute, raw string) (any, error) {
	var value any = raw
	switch attribute.DataType {
	case "number":
		number, err := strconv.ParseFloat(raw, 64)
		if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
			return nil, fmt.Errorf("custom attribute requires a number")
		}
		value = number
	case "checkbox":
		if raw != "true" && raw != "false" {
			return nil, fmt.Errorf("custom attribute requires true or false")
		}
		value = raw == "true"
	case "date":
		if _, err := time.Parse(time.DateOnly, raw); err != nil {
			return nil, fmt.Errorf("custom attribute requires a date: %w", err)
		}
	case "list":
		if !slices.Contains(attribute.Values, raw) {
			return nil, fmt.Errorf("custom attribute value is not an available option")
		}
	case "text", "link":
	default:
		return nil, fmt.Errorf("unsupported custom attribute type %q", attribute.DataType)
	}
	if attribute.Regex != "" {
		matched, err := regexp.MatchString(attribute.Regex, raw)
		if err != nil || !matched {
			return nil, fmt.Errorf("custom attribute value does not match its validation pattern")
		}
	}
	return value, nil
}
