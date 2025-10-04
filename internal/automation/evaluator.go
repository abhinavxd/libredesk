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
// If all the groups of a rule pass their evaluations based on the defined logical operations,
// the corresponding actions are executed.
func (e *Engine) evalConversationRules(rules []models.Rule, conversation cmodels.Conversation) {
	for _, rule := range rules {
		e.lo.Debug("evaluating rules for conversation", "rule", rule, "conversation_id", conversation.ID)

		// At max there can be only 2 groups.
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
			result := e.evaluateGroup(group.Rules, group.LogicalOp, conversation)
			e.lo.Debug("group rule evaluation complete", "logical_op", group.LogicalOp, "result", result, "conversation_uuid", conversation.UUID)
			groupEvalResults = append(groupEvalResults, result)
		}

		if evaluateFinalResult(groupEvalResults, rule.GroupOperator) {
			e.lo.Debug("all rules within groups evaluated successfully, executing actions", "conversation_uuid", conversation.UUID)
			for _, action := range rule.Actions {
				if err := e.conversationStore.ApplyAction(action, conversation, umodels.User{}); err != nil {
					e.lo.Error("error applying action on conversation", "action", action, "conversation_uuid", conversation.UUID, "error", err)
				}
			}
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
func (e *Engine) evaluateGroup(rules []models.RuleDetail, operator string, conversation cmodels.Conversation) bool {
	switch operator {
	case models.OperatorAnd:
		// All conditions within the group must be true
		for _, rule := range rules {
			if !e.evaluateRule(rule, conversation) {
				return false
			}
		}
		return true
	case models.OperatorOR:
		// At least one condition within the group must be true
		for _, rule := range rules {
			if e.evaluateRule(rule, conversation) {
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
func (e *Engine) evaluateRule(rule models.RuleDetail, conversation cmodels.Conversation) bool {
	var (
		valueToCompare   string
		ruleValues       []string
		conditionMet     bool
		customAttributes map[string]any
	)

	// Assign default field type if not provided for backward compatibility.
	if rule.FieldType == "" {
		rule.FieldType = models.FieldTypeConversationField
	}

	e.lo.Debug("evaluating rule", "rule_field", rule.Field, "field_type", rule.FieldType, "rule_operator", rule.Operator,
		"rule_value", rule.Value, "conversation_uuid", conversation.UUID)

	// Extract the value from the conversation based on the rule's field
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
		value1, _ := strconv.Atoi(valueToCompare)
		value2, _ := strconv.Atoi(rule.Value)
		conditionMet = value1 > value2
	case models.RuleOperatorLessThan:
		value1, _ := strconv.Atoi(valueToCompare)
		value2, _ := strconv.Atoi(rule.Value)
		conditionMet = value1 < value2
	default:
		e.lo.Error("error unrecognized rule logical operator", "operator", rule.Operator)
		return false
	}
	e.lo.Debug("conversation automation rule status", "has_met", conditionMet, "conversation_uuid", conversation.UUID)
	return conditionMet
}
