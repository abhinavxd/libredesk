package dbutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/abhinavxd/libredesk/internal/stringutil"
)

// ErrTooManyGroups is returned when a filter exceeds MaxFilterGroups. Callers map it to a user-facing error.
var ErrTooManyGroups = errors.New("too many filter groups")

var dateOnlyRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

var valueRequiredOperators = map[string]bool{
	"equals":       true,
	"not equals":   true,
	"greater than": true,
	"less than":    true,
	"in":           true,
	"between":      true,
	"contains":     true,
	"ilike":        true,
	"not contains": true,
}

// maxFilterDepth bounds group nesting (root group + one nested level).
const maxFilterDepth = 2

// MaxFilterGroups bounds how many groups a filter may contain (excluding the root).
const MaxFilterGroups = 10

// maxFilterConditions bounds how many leaf conditions a filter may contain in total.
const maxFilterConditions = 50

// maxInValues bounds how many values an "in" condition may carry.
const maxInValues = 100

// PaginationOptions represents the options for paginating a query.
type PaginationOptions struct {
	Page     int
	PageSize int
	OrderBy  string
	Order    string
	// Location is the IANA timezone date-only filters are resolved in; empty falls back to UTC.
	Location string
}

// Order directions.
const (
	ASC  = "ASC"
	DESC = "DESC"
)

// FilterNode is either a group (Logic + Rules) or a leaf (Model/Field/Operator/Value).
type FilterNode struct {
	Logic    string       `json:"logic,omitempty"`
	Rules    []FilterNode `json:"rules,omitempty"`
	Model    string       `json:"model,omitempty"`
	Field    string       `json:"field,omitempty"`
	Operator string       `json:"operator,omitempty"`
	Value    string       `json:"value,omitempty"`
}

func (n FilterNode) isGroup() bool {
	return n.Logic != "" || len(n.Rules) > 0
}

func (n FilterNode) isEmpty() bool {
	return !n.isGroup() && n.Field == ""
}

// AllowedFields is a map of model names to a list of allowed fields for that model.
type AllowedFields map[string][]string

// FieldRenderer renders a leaf condition for a field that does not map to a plain column,
// e.g. conversation tags rendered as a subquery. paramIndex is the next positional placeholder.
type FieldRenderer func(operator, value string, paramIndex int) (string, []any, error)

// FieldRenderers maps model -> field -> renderer.
type FieldRenderers map[string]map[string]FieldRenderer

func (r FieldRenderers) get(model, field string) (FieldRenderer, bool) {
	if r == nil {
		return nil, false
	}
	fields, ok := r[model]
	if !ok {
		return nil, false
	}
	fn, ok := fields[field]
	return fn, ok
}

// BuildPaginatedQuery builds a paginated query from the given base query, existing arguments, pagination options, filters JSON, allowed fields, and optional custom field renderers.
func BuildPaginatedQuery(baseQuery string, existingArgs []any, opts PaginationOptions, filtersJSON string, allowedFields AllowedFields, renderers FieldRenderers) (string, []any, error) {
	if opts.Page <= 0 {
		return "", nil, fmt.Errorf("invalid page number: %d", opts.Page)
	}
	if opts.PageSize <= 0 {
		return "", nil, fmt.Errorf("invalid page size: %d", opts.PageSize)
	}

	root, err := parseFilters(filtersJSON)
	if err != nil {
		return "", nil, err
	}

	loc := stringutil.NormalizeTimezone(opts.Location)

	whereClause, filterArgs, err := buildWhereClause(root, existingArgs, allowedFields, renderers, loc)
	if err != nil {
		return "", nil, err
	}

	query := baseQuery
	args := existingArgs

	if whereClause != "" {
		query += " AND " + whereClause
		args = append(args, filterArgs...)
	}

	if opts.OrderBy != "" {
		parts := strings.Split(opts.OrderBy, ".")
		if len(parts) != 2 {
			return "", nil, fmt.Errorf("invalid OrderBy format: %s", opts.OrderBy)
		}
		model, field := parts[0], parts[1]

		modelFields, ok := allowedFields[model]
		if !ok || !slices.Contains(modelFields, field) {
			return "", nil, fmt.Errorf("invalid OrderBy field: %s", opts.OrderBy)
		}

		order := strings.ToUpper(opts.Order)
		if order != "" && order != ASC && order != DESC {
			return "", nil, fmt.Errorf("invalid order direction: %s", opts.Order)
		}
		query += fmt.Sprintf(" ORDER BY %s.%s %s NULLS LAST", model, field, order)
	}

	offset := (opts.Page - 1) * opts.PageSize
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, opts.PageSize, offset)

	return query, args, nil
}

// ValidateFilters parses and structurally validates a filters payload without running a query.
func ValidateFilters(filtersJSON string, allowedFields AllowedFields, renderers FieldRenderers) error {
	root, err := parseFilters(filtersJSON)
	if err != nil {
		return err
	}
	args := []any{}
	next := 1
	_, err = buildNode(root, &args, &next, allowedFields, renderers, 0, "UTC")
	return err
}

// parseFilters accepts either a legacy flat array of leaves or a {logic, rules} group object.
func parseFilters(filtersJSON string) (FilterNode, error) {
	trimmed := strings.TrimSpace(filtersJSON)
	if trimmed == "" || trimmed == "[]" || trimmed == "null" {
		return FilterNode{}, nil
	}
	switch trimmed[0] {
	// OLD: flat array of leaves (legacy), implicitly AND-ed, e.g. [{"model":"conversations","field":"status_id","operator":"equals","value":"1"}]
	case '[':
		var leaves []FilterNode
		if err := json.Unmarshal([]byte(trimmed), &leaves); err != nil {
			return FilterNode{}, fmt.Errorf("invalid filters JSON: %w", err)
		}
		return FilterNode{Logic: "AND", Rules: leaves}, nil
	// NEW: logic + nested rules, e.g. {"logic":"AND","rules":[{"logic":"OR","rules":[<leaves>]},{"logic":"AND","rules":[<leaves>]}]}
	case '{':
		var node FilterNode
		if err := json.Unmarshal([]byte(trimmed), &node); err != nil {
			return FilterNode{}, fmt.Errorf("invalid filters JSON: %w", err)
		}
		return node, nil
	default:
		return FilterNode{}, fmt.Errorf("invalid filters JSON: expected array or object")
	}
}

func buildWhereClause(root FilterNode, existingArgs []any, allowedFields AllowedFields, renderers FieldRenderers, loc string) (string, []any, error) {
	args := []any{}
	next := len(existingArgs) + 1
	clause, err := buildNode(root, &args, &next, allowedFields, renderers, 0, loc)
	if err != nil {
		return "", nil, err
	}
	return clause, args, nil
}

func countGroups(node FilterNode) int {
	if !node.isGroup() {
		return 0
	}
	n := 1
	for _, child := range node.Rules {
		n += countGroups(child)
	}
	return n
}

func countConditions(node FilterNode) int {
	if !node.isGroup() {
		if node.isEmpty() {
			return 0
		}
		return 1
	}
	n := 0
	for _, child := range node.Rules {
		n += countConditions(child)
	}
	return n
}

func buildNode(node FilterNode, args *[]any, next *int, allowedFields AllowedFields, renderers FieldRenderers, depth int, loc string) (string, error) {
	if depth > maxFilterDepth {
		return "", fmt.Errorf("filter nesting too deep")
	}

	if depth == 0 {
		groups := 0
		for _, child := range node.Rules {
			groups += countGroups(child)
		}
		if groups > MaxFilterGroups {
			return "", ErrTooManyGroups
		}
		if countConditions(node) > maxFilterConditions {
			return "", fmt.Errorf("filter has too many conditions (max %d)", maxFilterConditions)
		}
	}

	if node.isEmpty() {
		return "", nil
	}

	if !node.isGroup() {
		return buildLeaf(node, args, next, allowedFields, renderers, loc)
	}

	logic := strings.ToUpper(strings.TrimSpace(node.Logic))
	if logic == "" {
		logic = "AND"
	}
	if logic != "AND" && logic != "OR" {
		return "", fmt.Errorf("invalid filter logic: %s", node.Logic)
	}

	parts := []string{}
	for _, child := range node.Rules {
		if child.isEmpty() {
			continue
		}
		clause, err := buildNode(child, args, next, allowedFields, renderers, depth+1, loc)
		if err != nil {
			return "", err
		}
		if clause != "" {
			parts = append(parts, clause)
		}
	}

	if len(parts) == 0 {
		return "", nil
	}
	return "(" + strings.Join(parts, " "+logic+" ") + ")", nil
}

func buildLeaf(f FilterNode, args *[]any, next *int, allowedFields AllowedFields, renderers FieldRenderers, loc string) (string, error) {
	if render, ok := renderers.get(f.Model, f.Field); ok {
		sql, rArgs, err := render(f.Operator, f.Value, *next)
		if err != nil {
			return "", err
		}
		*args = append(*args, rArgs...)
		*next += len(rArgs)
		return sql, nil
	}

	modelFields, ok := allowedFields[f.Model]
	if !ok {
		return "", fmt.Errorf("invalid model: %s", f.Model)
	}
	if !slices.Contains(modelFields, f.Field) {
		return "", fmt.Errorf("invalid field: %s for model: %s", f.Field, f.Model)
	}

	if valueRequiredOperators[f.Operator] && strings.TrimSpace(f.Value) == "" {
		return "", fmt.Errorf("operator %q requires a value", f.Operator)
	}

	field := fmt.Sprintf("%s.%s", f.Model, f.Field)

	switch f.Operator {
	case "equals":
		if dateOnlyRe.MatchString(f.Value) {
			cond := fmt.Sprintf("(%s >= ($%d::date)::timestamp AT TIME ZONE $%d AND %s < ($%d::date + 1)::timestamp AT TIME ZONE $%d)", field, *next, *next+1, field, *next, *next+1)
			*args = append(*args, f.Value, loc)
			*next += 2
			return cond, nil
		}
		cond := fmt.Sprintf("%s = $%d", field, *next)
		*args = append(*args, f.Value)
		*next++
		return cond, nil
	case "not equals":
		if dateOnlyRe.MatchString(f.Value) {
			cond := fmt.Sprintf("(%s < ($%d::date)::timestamp AT TIME ZONE $%d OR %s >= ($%d::date + 1)::timestamp AT TIME ZONE $%d)", field, *next, *next+1, field, *next, *next+1)
			*args = append(*args, f.Value, loc)
			*next += 2
			return cond, nil
		}
		cond := fmt.Sprintf("%s != $%d", field, *next)
		*args = append(*args, f.Value)
		*next++
		return cond, nil
	case "greater than":
		if dateOnlyRe.MatchString(f.Value) {
			cond := fmt.Sprintf("%s >= ($%d::date + 1)::timestamp AT TIME ZONE $%d", field, *next, *next+1)
			*args = append(*args, f.Value, loc)
			*next += 2
			return cond, nil
		}
		cond := fmt.Sprintf("%s > $%d", field, *next)
		*args = append(*args, f.Value)
		*next++
		return cond, nil
	case "less than":
		if dateOnlyRe.MatchString(f.Value) {
			cond := fmt.Sprintf("%s < ($%d::date)::timestamp AT TIME ZONE $%d", field, *next, *next+1)
			*args = append(*args, f.Value, loc)
			*next += 2
			return cond, nil
		}
		cond := fmt.Sprintf("%s < $%d", field, *next)
		*args = append(*args, f.Value)
		*next++
		return cond, nil
	case "set":
		return field + " IS NOT NULL", nil
	case "not set":
		return field + " IS NULL", nil
	case "in":
		var arr []string
		if err := json.Unmarshal([]byte(f.Value), &arr); err != nil {
			return "", fmt.Errorf("invalid array format for 'in' operator: %v", err)
		}
		if len(arr) == 0 {
			return "", fmt.Errorf("operator \"in\" requires at least one value")
		}
		if len(arr) > maxInValues {
			return "", fmt.Errorf("operator \"in\" allows at most %d values", maxInValues)
		}
		placeholders := make([]string, len(arr))
		for i, v := range arr {
			placeholders[i] = fmt.Sprintf("$%d", *next)
			*args = append(*args, v)
			*next++
		}
		return field + " IN (" + strings.Join(placeholders, ",") + ")", nil
	case "between":
		values := strings.Split(f.Value, ",")
		if len(values) != 2 {
			return "", fmt.Errorf("between requires 2 values")
		}
		start := strings.TrimSpace(values[0])
		end := strings.TrimSpace(values[1])
		if dateOnlyRe.MatchString(start) && dateOnlyRe.MatchString(end) {
			cond := fmt.Sprintf("(%s >= ($%d::date)::timestamp AT TIME ZONE $%d AND %s < ($%d::date + 1)::timestamp AT TIME ZONE $%d)", field, *next, *next+2, field, *next+1, *next+2)
			*args = append(*args, start, end, loc)
			*next += 3
			return cond, nil
		}
		cond := fmt.Sprintf("%s BETWEEN $%d AND $%d", field, *next, *next+1)
		*args = append(*args, start, end)
		*next += 2
		return cond, nil
	case "contains", "ilike":
		cond := fmt.Sprintf("%s ILIKE $%d", field, *next)
		*args = append(*args, "%"+f.Value+"%")
		*next++
		return cond, nil
	case "not contains":
		cond := fmt.Sprintf("%s NOT ILIKE $%d", field, *next)
		*args = append(*args, "%"+f.Value+"%")
		*next++
		return cond, nil
	default:
		return "", fmt.Errorf("invalid operator: %s", f.Operator)
	}
}
