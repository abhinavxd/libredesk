package automation

import (
	"encoding/json"
	"fmt"
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
					e.lo.Error("error applying action on conversation", "action", action, "conversation_uuid", conversation.UUID, "error", err)
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
		case models.ConversationSubject:
			valueToCompare = conversation.Subject.String
		case models.ConversationContent:
			valueToCompare = conversation.LastMessage.String
		case models.ConversationStatus:
			valueToCompare = strconv.Itoa(conversation.StatusID.Int)
		case models.ConversationPriority:
			valueToCompare = strconv.Itoa(conversation.PriorityID.Int)
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
			return evaluateStringValues(conversation.IncomingTo, rule)
		default:
			e.lo.Error("error unrecognized conversation field", "field", rule.Field, "field_type", rule.FieldType, "conversation_uuid", conversation.UUID)
			return false
		}
	} else if rule.FieldType == models.FieldTypeContactCustomAttribute {
		// If the field type is custom attribute, need to extract the value from the custom attributes
		var attributes json.RawMessage = conversation.Contact.CustomAttributes

		// Unmarshal the custom attributes
		if err := json.Unmarshal(attributes, &customAttributes); err != nil {
			e.lo.Error("error unmarshalling custom attributes", "conversation_uuid", conversation.UUID, "error", err)
			return false
		}
		e.lo.Debug("unmarshalled custom attributes", "custom_attributes", customAttributes, "conversation_uuid", conversation.UUID)

		// Check if the field exists in the custom attributes, If the field is not found, return false.
		if val, ok := customAttributes[rule.Field]; ok {
			// Convert the value to a string for comparison, Handle different types of values, really not required but just to be safe.
			switch v := val.(type) {
			case string:
				valueToCompare = v
			case int:
				valueToCompare = strconv.Itoa(v)
			// Float type does not exist in the custom attributes.
			case float64:
				valueToCompare = strconv.FormatInt(int64(v), 10)
			case bool:
				valueToCompare = strconv.FormatBool(v)
			default:
				valueToCompare = fmt.Sprintf("%v", v)
			}
		} else {
			e.lo.Warn("field not found in custom attribute", "field", rule.Field, "field_type", rule.FieldType, "conversation_uuid", conversation.UUID, "custom_attributes", customAttributes)
			return false
		}
	} else {
		e.lo.Error("error unrecognized field type", "field_type", rule.FieldType, "conversation_uuid", conversation.UUID)
		return false
	}

	conditionMet := evaluateValue(valueToCompare, rule)
	e.lo.Debug("conversation automation rule status", "has_met", conditionMet, "conversation_uuid", conversation.UUID)
	return conditionMet
}

// evaluateValue compares one field value against a rule.
func evaluateValue(value string, rule models.RuleDetail) bool {
	left, right := value, rule.Value
	if !rule.CaseSensitiveMatch {
		left, right = strings.ToLower(left), strings.ToLower(right)
	}

	switch rule.Operator {
	case models.RuleOperatorEquals:
		return left == right
	case models.RuleOperatorNotEqual:
		return left != right
	case models.RuleOperatorContains, models.RuleOperatorNotContains:
		left = strings.Join(strings.Fields(left), " ")
		matches := false
		for _, ruleValue := range strings.Split(right, ",") {
			ruleValue = strings.Join(strings.Fields(ruleValue), " ")
			if strings.Contains(left, ruleValue) {
				matches = true
				break
			}
		}
		return matches == (rule.Operator == models.RuleOperatorContains)
	case models.RuleOperatorSet:
		return len(value) > 0
	case models.RuleOperatorNotSet:
		return len(value) == 0
	case models.RuleOperatorGreaterThan:
		value1, _ := strconv.Atoi(value)
		value2, _ := strconv.Atoi(rule.Value)
		return value1 > value2
	case models.RuleOperatorLessThan:
		value1, _ := strconv.Atoi(value)
		value2, _ := strconv.Atoi(rule.Value)
		return value1 < value2
	case models.RuleOperatorStartsWith:
		return strings.HasPrefix(left, right)
	default:
		return false
	}
}

// evaluateStringValues compares a rule against a multi-value field such as email recipients.
// Positive operators match any value; negative operators require every value to differ.
func evaluateStringValues(values []string, rule models.RuleDetail) bool {
	nonEmptyValues := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			nonEmptyValues = append(nonEmptyValues, value)
		}
	}
	values = nonEmptyValues

	if rule.Operator == models.RuleOperatorSet {
		return len(values) > 0
	}
	if rule.Operator == models.RuleOperatorNotSet {
		return len(values) == 0
	}

	negative := rule.Operator == models.RuleOperatorNotEqual || rule.Operator == models.RuleOperatorNotContains
	if rule.Operator == models.RuleOperatorNotEqual {
		rule.Operator = models.RuleOperatorEquals
	} else if rule.Operator == models.RuleOperatorNotContains {
		rule.Operator = models.RuleOperatorContains
	}

	for _, value := range values {
		if evaluateValue(value, rule) {
			return !negative
		}
	}
	return negative
}
