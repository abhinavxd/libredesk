package automation

import (
	"cmp"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/automation/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
)

// evalConversationRules evaluates a list of rules against a given conversation.
func (e *Engine) evalConversationRules(rules []models.Rule, conversation cmodels.Conversation, previousValues map[string]string) {
	for _, rule := range rules {
		e.lo.Debug("evaluating rules for conversation", "rule", rule, "conversation_id", conversation.ID)

		if len(rule.Groups) > 2 {
			e.lo.Warn("WARNING: more than 2 groups found for rules skipping evaluation")
			continue
		}

		var groupEvalResults []bool
		for idx, group := range rule.Groups {
			if len(group.Rules) == 0 {
				e.lo.Debug("no rules found in group, skipping rule group evaluation", "group_num", idx+1, "conversation_uuid", conversation.UUID)
				continue
			}
			result := e.evaluateGroup(group.Rules, group.LogicalOp, conversation, previousValues)
			e.lo.Debug("group rule evaluation complete", "logical_op", group.LogicalOp, "result", result, "conversation_uuid", conversation.UUID)
			groupEvalResults = append(groupEvalResults, result)
		}

		if evaluateFinalResult(groupEvalResults, rule.GroupOperator) {
			e.lo.Debug("all rules within groups evaluated successfully, executing actions", "conversation_uuid", conversation.UUID)
			e.suppress(conversation.UUID)
			for _, action := range rule.Actions {
				if err := e.conversationStore.ApplyAction(action, conversation, umodels.User{}); err != nil {
					e.lo.Error("error applying action on conversation", "rule_id", rule.ID, "action", action, "conversation_uuid", conversation.UUID, "error", err)
				} else {
					e.lo.Info("automation action completed", "rule_id", rule.ID, "action", action.Type, "conversation_uuid", conversation.UUID)
				}
			}
			e.unsuppress(conversation.UUID)
			if rule.ExecutionMode == models.ExecutionModeFirstMatch {
				e.lo.Debug("automation is first match rule execution mode, breaking out of rule evaluation", "conversation_uuid", conversation.UUID)
				break
			}
		} else {
			e.lo.Debug("rule evaluation failed, skipping actions", "group_eval_results", groupEvalResults, "conversation_uuid", conversation.UUID)
		}
	}
}

// evaluateFinalResult computes the final result of multiple group evaluations
// based on the specified logical operator (AND/OR).
func evaluateFinalResult(results []bool, operator string) bool {
	if operator == models.OperatorAnd {
		for _, result := range results {
			if !result {
				return false
			}
		}
		return true
	}
	if operator == models.OperatorOR {
		for _, result := range results {
			if result {
				return true
			}
		}
		return false
	}
	return false
}

// evaluateGroup evaluates a set of rules within a group against a given conversation
// based on the specified logical operator (AND/OR).
func (e *Engine) evaluateGroup(rules []models.RuleDetail, operator string, conversation cmodels.Conversation, previousValues map[string]string) bool {
	switch operator {
	case models.OperatorAnd:
		for _, rule := range rules {
			if !e.evaluateRule(rule, conversation, previousValues) {
				return false
			}
		}
		return true
	case models.OperatorOR:
		for _, rule := range rules {
			if e.evaluateRule(rule, conversation, previousValues) {
				return true
			}
		}
		return false
	default:
		e.lo.Error("invalid group operator", "operator", operator)
	}
	return false
}

// evaluateRule evaluates a single rule against a given conversation by extracting the field value and comparing it with the rule's value.
// Returns true if the rule condition is met, false otherwise.
func (e *Engine) evaluateRule(rule models.RuleDetail, conversation cmodels.Conversation, previousValues map[string]string) bool {
	var (
		valueToCompare   string
		ruleValues       []string
		conditionMet     bool
		customAttributes map[string]any
	)

	if rule.FieldType == "" {
		rule.FieldType = models.FieldTypeConversationField
	}

	e.lo.Debug("evaluating rule", "rule_field", rule.Field, "field_type", rule.FieldType, "rule_operator", rule.Operator,
		"rule_value", rule.Value, "conversation_uuid", conversation.UUID)

	if rule.FieldType == models.FieldTypeConversationField {
		switch rule.Field {
		case models.ContactEmail:
			valueToCompare = conversation.Contact.Email.String
			// Email addresses are case-insensitive, older rules may still carry the flag.
			rule.CaseSensitiveMatch = false
		case models.ConversationSubject:
			valueToCompare = conversation.Subject.String
		case models.ConversationContent:
			valueToCompare = conversation.LastMessage.String
		case models.ConversationStatus:
			valueToCompare = strconv.Itoa(conversation.StatusID.Int)
		case models.ConversationPriority:
			valueToCompare = strconv.Itoa(conversation.PriorityID.Int)
		case models.ConversationTags:
			var tags []string
			if len(conversation.Tags.JSON) > 0 {
				if err := json.Unmarshal(conversation.Tags.JSON, &tags); err != nil {
					e.lo.Error("error parsing conversation tags", "error", err)
					return false
				}
			}
			return evaluateTagRule(tags, rule)
		case models.ConversationChannel:
			valueToCompare = conversation.InboxChannel
		case models.ConversationStatusCategory:
			valueToCompare = conversation.StatusCategory.String
		case models.ConversationLastMessageSender:
			valueToCompare = conversation.LastInteractionSender.String
		case models.ContactCountry:
			valueToCompare = conversation.Contact.Country.String
		case models.ContactExternalID:
			valueToCompare = conversation.Contact.ExternalUserID.String
		case models.ContactPhone:
			valueToCompare = conversation.Contact.PhoneNumber.String
		case models.ContactType:
			valueToCompare = conversation.Contact.Type
		case models.ConversationSLAPolicy:
			if conversation.SLAPolicyID.Valid {
				valueToCompare = strconv.Itoa(conversation.SLAPolicyID.Int)
			}
		case models.ConversationSLABreached:
			valueToCompare = strconv.FormatBool(conversation.SLABreached)
		case models.ConversationHoursUntilSLADeadline:
			if conversation.NextSLADeadlineAt.Valid {
				valueToCompare = strconv.FormatFloat(time.Until(conversation.NextSLADeadlineAt.Time).Hours(), 'f', -1, 64)
			}
		case models.ConversationCSATRating:
			if conversation.CSATRespondedAt.Valid && conversation.CSATRating.Valid && conversation.CSATRating.Int > 0 {
				valueToCompare = strconv.Itoa(conversation.CSATRating.Int)
			}
		case models.ConversationCSATFeedback:
			if conversation.CSATRespondedAt.Valid {
				valueToCompare = conversation.CSATFeedback.String
			}
		case models.ConversationWithinBusinessHours:
			if e.businessHoursChecker == nil {
				return false
			}
			within, err := e.businessHoursChecker(conversation.AssignedTeamID.Int, time.Now())
			if err != nil {
				e.lo.Error("error checking business hours", "error", err, "conversation_uuid", conversation.UUID)
				return false
			}
			valueToCompare = strconv.FormatBool(within)
		case models.ConversationHoursWaiting, models.ConversationHoursSinceLastMessage,
			models.ConversationHoursSinceLastIncoming, models.ConversationHoursSinceLastActivity:
			var at time.Time
			switch rule.Field {
			case models.ConversationHoursWaiting:
				at = conversation.WaitingSince.Time
			case models.ConversationHoursSinceLastMessage:
				at = conversation.LastInteractionAt.Time
			case models.ConversationHoursSinceLastIncoming:
				at = conversation.LastInboundAt.Time
			case models.ConversationHoursSinceLastActivity:
				at = conversation.LastMessageAt.Time
			}
			if !at.IsZero() {
				valueToCompare = strconv.FormatFloat(time.Since(at).Hours(), 'f', -1, 64)
			}
		case models.ConversationAssignedTeam:
			if conversation.AssignedTeamID.Valid {
				valueToCompare = strconv.Itoa(conversation.AssignedTeamID.Int)
			}
		case models.ConversationAssignedUser:
			if conversation.AssignedUserID.Valid {
				valueToCompare = strconv.Itoa(conversation.AssignedUserID.Int)
			}
		case models.ConversationHoursSinceCreated:
			valueToCompare = fmt.Sprintf("%.0f", (time.Since(conversation.CreatedAt).Hours()))
		case models.ConversationHoursSinceFirstReply:
			if !conversation.FirstReplyAt.IsZero() {
				valueToCompare = fmt.Sprintf("%.0f", (time.Since(conversation.FirstReplyAt.Time).Hours()))
			}
		case models.ConversationHoursSinceLastReply:
			if !conversation.LastReplyAt.IsZero() {
				valueToCompare = fmt.Sprintf("%.0f", (time.Since(conversation.LastReplyAt.Time).Hours()))
			}
		case models.ConversationHoursSinceResolved:
			if !conversation.ResolvedAt.IsZero() {
				valueToCompare = fmt.Sprintf("%.0f", (time.Since(conversation.ResolvedAt.Time).Hours()))
			}
		case models.ConversationHoursSinceLastResolved:
			if !conversation.LastResolvedAt.IsZero() {
				valueToCompare = fmt.Sprintf("%.0f", time.Since(conversation.LastResolvedAt.Time).Hours())
			} else if rule.Operator != models.RuleOperatorSet && rule.Operator != models.RuleOperatorNotSet {
				return false
			}
		case models.ConversationInbox:
			valueToCompare = strconv.Itoa(conversation.InboxID)
		case models.ConversationPreviousStatus, models.ConversationPreviousPriority,
			models.ConversationPreviousAssignedUser, models.ConversationPreviousAssignedTeam:
			// An absent key is not the same as an empty previous value.
			previous, ok := previousValues[rule.Field]
			if !ok {
				e.lo.Debug("no previous value available for field, skipping rule", "field", rule.Field, "conversation_uuid", conversation.UUID)
				return false
			}
			valueToCompare = previous
		case models.ConversationIncomingTo:
			return evaluateRecipientRule(conversation.IncomingTo, rule)
		default:
			e.lo.Error("error unrecognized conversation field", "field", rule.Field, "field_type", rule.FieldType, "conversation_uuid", conversation.UUID)
			return false
		}
	} else if rule.FieldType == models.FieldTypeContactCustomAttribute || rule.FieldType == models.FieldTypeConversationCustomAttribute {
		// If the field type is custom attribute, need to extract the value from the custom attributes
		var attributes json.RawMessage = conversation.Contact.CustomAttributes
		if rule.FieldType == models.FieldTypeConversationCustomAttribute {
			attributes = conversation.CustomAttributes
		}
		if len(attributes) == 0 {
			attributes = json.RawMessage(`{}`)
		}

		// Unmarshal the custom attributes
		if err := json.Unmarshal(attributes, &customAttributes); err != nil {
			e.lo.Error("error unmarshalling custom attributes", "conversation_uuid", conversation.UUID, "error", err)
			return false
		}
		e.lo.Debug("unmarshalled custom attributes", "custom_attributes", customAttributes, "conversation_uuid", conversation.UUID)

		// Only the set/not set operators can act on an attribute the contact does not have.
		val, ok := customAttributes[rule.Field]
		if val == nil {
			ok = false
		}
		if !ok && rule.Operator != models.RuleOperatorSet && rule.Operator != models.RuleOperatorNotSet {
			e.lo.Warn("field not found in custom attribute", "field", rule.Field, "field_type", rule.FieldType, "conversation_uuid", conversation.UUID)
			return false
		}
		if ok {
			// Convert the value to a string for comparison, Handle different types of values, really not required but just to be safe.
			switch v := val.(type) {
			case string:
				valueToCompare = v
			case int:
				valueToCompare = strconv.Itoa(v)
			case float64:
				valueToCompare = strconv.FormatFloat(v, 'f', -1, 64)
			case bool:
				valueToCompare = strconv.FormatBool(v)
			default:
				valueToCompare = fmt.Sprintf("%v", v)
			}
		}
	} else {
		e.lo.Error("error unrecognized field type", "field_type", rule.FieldType, "conversation_uuid", conversation.UUID)
		return false
	}

	// Case sensitive match?
	if !rule.CaseSensitiveMatch {
		valueToCompare = strings.ToLower(valueToCompare)
		rule.Value = strings.ToLower(rule.Value)
	}

	// Split and trim values for Contains/NotContains operations
	if rule.Operator == models.RuleOperatorContains || rule.Operator == models.RuleOperatorNotContains {
		ruleValues = strings.Split(rule.Value, ",")
		for i := range ruleValues {
			ruleValues[i] = strings.TrimSpace(ruleValues[i])
			if !rule.CaseSensitiveMatch {
				ruleValues[i] = strings.ToLower(ruleValues[i])
			}
		}
	}

	e.lo.Debug("evaluating rule", "rule_field", rule.Field, "rule_operator", rule.Operator,
		"rule_value", rule.Value, "rule_values", ruleValues, "value_to_compare",
		valueToCompare, "conversation_uuid", conversation.UUID)

	// Compare with set operator
	switch rule.Operator {
	case models.RuleOperatorEquals:
		conditionMet = valueToCompare == rule.Value
	case models.RuleOperatorNotEqual:
		conditionMet = valueToCompare != rule.Value
	case models.RuleOperatorContains:
		// Normalize input text by collapsing multiple spaces
		normalizedInputText := strings.Join(strings.Fields(valueToCompare), " ")
		conditionMet = false

		// Check each rule value against the normalized input
		for _, ruleValue := range ruleValues {
			// Normalize rule value by collapsing multiple spaces
			normalizedRuleValue := strings.Join(strings.Fields(ruleValue), " ")

			// Respect CaseSensitiveMatch flag
			if rule.CaseSensitiveMatch {
				if strings.Contains(normalizedInputText, normalizedRuleValue) {
					conditionMet = true
					break
				}
			} else {
				if strings.Contains(
					strings.ToLower(normalizedInputText),
					strings.ToLower(normalizedRuleValue),
				) {
					conditionMet = true
					break
				}
			}
		}
	case models.RuleOperatorNotContains:
		// Normalize input text by collapsing multiple spaces
		normalizedInputText := strings.Join(strings.Fields(valueToCompare), " ")
		conditionMet = true

		// Check each rule value against the normalized input
		for _, ruleValue := range ruleValues {
			// Normalize rule value by collapsing multiple spaces
			normalizedRuleValue := strings.Join(strings.Fields(ruleValue), " ")

			// Respect CaseSensitiveMatch flag
			if rule.CaseSensitiveMatch {
				if strings.Contains(normalizedInputText, normalizedRuleValue) {
					conditionMet = false
					break
				}
			} else {
				if strings.Contains(
					strings.ToLower(normalizedInputText),
					strings.ToLower(normalizedRuleValue),
				) {
					conditionMet = false
					break
				}
			}
		}
	case models.RuleOperatorSet:
		conditionMet = len(valueToCompare) > 0
	case models.RuleOperatorNotSet:
		conditionMet = len(valueToCompare) == 0
	case models.RuleOperatorGreaterThan:
		comparison, ok := compareRuleValues(valueToCompare, rule.Value)
		conditionMet = ok && comparison > 0
	case models.RuleOperatorLessThan:
		comparison, ok := compareRuleValues(valueToCompare, rule.Value)
		conditionMet = ok && comparison < 0
	case models.RuleOperatorStartsWith:
		conditionMet = strings.HasPrefix(valueToCompare, rule.Value)
	default:
		e.lo.Error("error unrecognized rule logical operator", "operator", rule.Operator)
		return false
	}
	e.lo.Debug("conversation automation rule status", "has_met", conditionMet, "conversation_uuid", conversation.UUID)
	return conditionMet
}

// Tag conditions compare whole names, including names containing commas.
func evaluateTagRule(tags []string, rule models.RuleDetail) bool {
	if rule.Operator == models.RuleOperatorSet {
		return len(tags) > 0
	}
	if rule.Operator == models.RuleOperatorNotSet {
		return len(tags) == 0
	}
	if rule.Operator != models.RuleOperatorContains && rule.Operator != models.RuleOperatorNotContains {
		return false
	}
	var values []string
	if err := json.Unmarshal([]byte(rule.Value), &values); err != nil || len(values) == 0 {
		return false
	}
	matched := slices.ContainsFunc(values, func(value string) bool { return slices.Contains(tags, value) })
	if rule.Operator == models.RuleOperatorNotContains {
		return !matched
	}
	return matched
}

func compareRuleValues(left, right string) (int, bool) {
	a, errA := strconv.ParseFloat(left, 64)
	b, errB := strconv.ParseFloat(right, 64)
	if errA == nil && errB == nil && !math.IsNaN(a) && !math.IsNaN(b) && !math.IsInf(a, 0) && !math.IsInf(b, 0) {
		return cmp.Compare(a, b), true
	}
	dateA, errA := time.Parse(time.DateOnly, left)
	dateB, errB := time.Parse(time.DateOnly, right)
	if errA == nil && errB == nil {
		return dateA.Compare(dateB), true
	}
	return 0, false
}

func evaluateRecipientRule(recipients []string, rule models.RuleDetail) bool {
	negative := rule.Operator == models.RuleOperatorNotEqual || rule.Operator == models.RuleOperatorNotContains
	ruleValue := strings.ToLower(strings.TrimSpace(rule.Value))

	for _, recipient := range recipients {
		recipient = strings.TrimSpace(recipient)
		if recipient == "" {
			continue
		}
		if rule.Operator == models.RuleOperatorSet {
			return true
		}
		if rule.Operator == models.RuleOperatorNotSet {
			return false
		}
		recipient = strings.ToLower(recipient)

		var matched bool
		switch rule.Operator {
		case models.RuleOperatorEquals, models.RuleOperatorNotEqual:
			matched = recipient == ruleValue
		case models.RuleOperatorContains, models.RuleOperatorNotContains:
			recipient = strings.Join(strings.Fields(recipient), " ")
			for candidate := range strings.SplitSeq(ruleValue, ",") {
				candidate = strings.Join(strings.Fields(candidate), " ")
				if candidate == "" {
					continue
				}
				if strings.Contains(recipient, candidate) {
					matched = true
					break
				}
			}
		default:
			return false
		}
		if matched {
			return !negative
		}
	}

	if rule.Operator == models.RuleOperatorSet {
		return false
	}
	if rule.Operator == models.RuleOperatorNotSet {
		return true
	}
	return negative
}
