package proactive

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"

	amodels "github.com/abhinavxd/libredesk/internal/automation/models"
	"github.com/google/uuid"
)

var campaignOperators = []string{amodels.RuleOperatorContains, amodels.RuleOperatorNotContains, amodels.RuleOperatorEquals, amodels.RuleOperatorNotEqual, amodels.RuleOperatorSet, amodels.RuleOperatorNotSet, amodels.RuleOperatorGreaterThan, amodels.RuleOperatorLessThan, amodels.RuleOperatorStartsWith}

func (c Campaign) Validate() error {
	if _, err := uuid.Parse(c.ID); err != nil {
		return fmt.Errorf("id")
	}
	if strings.TrimSpace(c.Name) == "" || len(c.Name) > 128 {
		return fmt.Errorf("name")
	}
	if strings.TrimSpace(c.Message) == "" || len(c.Message) > 10000 {
		return fmt.Errorf("message")
	}
	if !slices.Contains([]string{"all", "visitors", "users"}, c.Audience) {
		return fmt.Errorf("audience")
	}
	if !slices.Contains([]string{"once", "session", "interval"}, c.Repeat) {
		return fmt.Errorf("repeat")
	}
	if !slices.Contains([]string{"any", "inside", "outside"}, c.BusinessHours) {
		return fmt.Errorf("business_hours")
	}
	if c.SenderID < 0 || c.TeamID < 0 || c.BusinessHoursID < 0 || (c.BusinessHours != "any" && c.BusinessHoursID == 0) {
		return fmt.Errorf("sender, team or business hours")
	}
	if !c.Desktop && !c.Mobile {
		return fmt.Errorf("device")
	}
	if c.DelaySeconds < 0 || c.DelaySeconds > 86400 || c.RepeatHours < 1 || c.RepeatHours > 8760 {
		return fmt.Errorf("delay or repeat interval")
	}
	if len(c.Event) > 128 || len(c.IncludeURLs)+len(c.ExcludeURLs) > 40 {
		return fmt.Errorf("targeting")
	}
	for _, pattern := range append(slices.Clone(c.IncludeURLs), c.ExcludeURLs...) {
		if len(pattern) > 2048 || strings.TrimSpace(pattern) == "" {
			return fmt.Errorf("url pattern")
		}
	}
	return ValidateConditions(c.Conditions)
}

func (c Campaign) IneligibleReason(ctx Context, conditionsMatch bool, withinHours func() bool) string {
	switch {
	case !c.Enabled:
		return "paused"
	case c.Audience == "visitors" && !ctx.Visitor || c.Audience == "users" && ctx.Visitor:
		return "audience"
	case ctx.Mobile && !c.Mobile || !ctx.Mobile && !c.Desktop:
		return "device"
	case !conditionsMatch:
		return "attributes"
	case len(c.IncludeURLs) > 0 && !matchesURL(c.IncludeURLs, ctx.URL):
		return "url"
	case matchesURL(c.ExcludeURLs, ctx.URL):
		return "excludedUrl"
	case c.Event != "" && c.Event != ctx.Event:
		return "event"
	case ctx.ActiveSeconds < c.DelaySeconds:
		return "delay"
	case c.BusinessHours == "inside" && !withinHours() || c.BusinessHours == "outside" && withinHours():
		return "businessHours"
	}
	return ""
}

func ValidateConditions(group amodels.RuleGroup) error {
	if len(group.Rules) > 20 || len(group.Rules) > 0 && group.LogicalOp != "AND" && group.LogicalOp != "OR" {
		return fmt.Errorf("conditions")
	}
	for _, rule := range group.Rules {
		if rule.FieldType != amodels.FieldTypeContactCustomAttribute || rule.Field == "" || len(rule.Value) > 2048 || !slices.Contains(campaignOperators, rule.Operator) {
			return fmt.Errorf("conditions")
		}
	}
	return nil
}

func matchesURL(patterns []string, raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.Scheme != "https" && u.Scheme != "http" {
		return false
	}
	u.Fragment = ""
	for _, pattern := range patterns {
		target := *u
		if !strings.Contains(pattern, "?") {
			target.RawQuery = ""
		}
		candidate := target.String()
		switch {
		case strings.HasPrefix(pattern, "/"):
			candidate = target.RequestURI()
		case !strings.Contains(pattern, "://"):
			candidate = strings.TrimPrefix(candidate, target.Scheme+"://")
		}
		expression := "^" + strings.ReplaceAll(regexp.QuoteMeta(pattern), `\*`, ".*") + "$"
		if regexp.MustCompile(expression).MatchString(candidate) {
			return true
		}
	}
	return false
}
