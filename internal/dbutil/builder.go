package dbutil

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

var dateOnlyRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// PaginationOptions represents the options for paginating a query.
type PaginationOptions struct {
	Page     int
	PageSize int
	OrderBy  string
	Order    string
}

// Order directions.
const (
	ASC  = "ASC"
	DESC = "DESC"
)

// Filter represents a filter to be applied to a query.
type Filter struct {
	Model    string `json:"model"`
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

// AllowedFields is a map of model names to a list of allowed fields for that model.
type AllowedFields map[string][]string

// BuildPaginatedQuery builds a paginated query from the given base query, existing arguments, pagination options, filters JSON, and allowed fields.
func BuildPaginatedQuery(baseQuery string, existingArgs []any, opts PaginationOptions, filtersJSON string, allowedFields AllowedFields) (string, []any, error) {
	if opts.Page <= 0 {
		return "", nil, fmt.Errorf("invalid page number: %d", opts.Page)
	}
	if opts.PageSize <= 0 {
		return "", nil, fmt.Errorf("invalid page size: %d", opts.PageSize)
	}

	var filters []Filter
	if filtersJSON != "" {
		if err := json.Unmarshal([]byte(filtersJSON), &filters); err != nil {
			return "", nil, fmt.Errorf("invalid filters JSON: %w", err)
		}
	}

	whereClause, filterArgs, err := buildWhereClause(filters, existingArgs, allowedFields)
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
		// Validate OrderBy.
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

// buildWhereClause builds a WHERE clause from the given filters and returns the WHERE clause and the arguments to be passed to the query.
func buildWhereClause(filters []Filter, existingArgs []interface{}, allowedFields AllowedFields) (string, []interface{}, error) {
	conditions := []string{}
	args := []interface{}{}
	paramCount := len(existingArgs) + 1

	for _, f := range filters {
		modelFields, ok := allowedFields[f.Model]
		if !ok {
			return "", nil, fmt.Errorf("invalid model: %s", f.Model)
		}
		if !slices.Contains(modelFields, f.Field) {
			return "", nil, fmt.Errorf("invalid field: %s for model: %s", f.Field, f.Model)
		}

		field := fmt.Sprintf("%s.%s", f.Model, f.Field)

		switch f.Operator {
		case "equals":
			if dateOnlyRe.MatchString(f.Value) {
				conditions = append(conditions, fmt.Sprintf("%s >= $%d::DATE AND %s < ($%d::DATE + INTERVAL '1 day')", field, paramCount, field, paramCount))
				args = append(args, f.Value)
				paramCount++
				break
			}
			conditions = append(conditions, field+fmt.Sprintf(" = $%d", paramCount))
			args = append(args, f.Value)
			paramCount++
		case "not equals":
			if dateOnlyRe.MatchString(f.Value) {
				conditions = append(conditions, fmt.Sprintf("(%s < $%d::DATE OR %s >= ($%d::DATE + INTERVAL '1 day'))", field, paramCount, field, paramCount))
				args = append(args, f.Value)
				paramCount++
				break
			}
			conditions = append(conditions, field+fmt.Sprintf(" != $%d", paramCount))
			args = append(args, f.Value)
			paramCount++
		case "greater than":
			if dateOnlyRe.MatchString(f.Value) {
				conditions = append(conditions, fmt.Sprintf("%s >= ($%d::DATE + INTERVAL '1 day')", field, paramCount))
				args = append(args, f.Value)
				paramCount++
				break
			}
			conditions = append(conditions, field+fmt.Sprintf(" > $%d", paramCount))
			args = append(args, f.Value)
			paramCount++
		case "less than":
			if dateOnlyRe.MatchString(f.Value) {
				conditions = append(conditions, fmt.Sprintf("%s < $%d::DATE", field, paramCount))
				args = append(args, f.Value)
				paramCount++
				break
			}
			conditions = append(conditions, field+fmt.Sprintf(" < $%d", paramCount))
			args = append(args, f.Value)
			paramCount++
		case "set":
			conditions = append(conditions, field+" IS NOT NULL")
		case "not set":
			conditions = append(conditions, field+" IS NULL")
		case "in":
			var arr []string
			if err := json.Unmarshal([]byte(f.Value), &arr); err != nil {
				return "", nil, fmt.Errorf("invalid array format for 'in' operator: %v", err)
			}
			placeholders := make([]string, len(arr))
			for i, v := range arr {
				placeholders[i] = fmt.Sprintf("$%d", paramCount)
				args = append(args, v)
				paramCount++
			}
			conditions = append(conditions, field+" IN ("+strings.Join(placeholders, ",")+")")
		case "between":
			values := strings.Split(f.Value, ",")
			if len(values) != 2 {
				return "", nil, fmt.Errorf("between requires 2 values")
			}
			start := strings.TrimSpace(values[0])
			end := strings.TrimSpace(values[1])
			if dateOnlyRe.MatchString(start) && dateOnlyRe.MatchString(end) {
				conditions = append(conditions, fmt.Sprintf("%s >= $%d::DATE AND %s < ($%d::DATE + INTERVAL '1 day')", field, paramCount, field, paramCount+1))
			} else {
				conditions = append(conditions, fmt.Sprintf("%s BETWEEN $%d AND $%d", field, paramCount, paramCount+1))
			}
			args = append(args, start, end)
			paramCount += 2
		case "ilike":
			conditions = append(conditions, field+fmt.Sprintf(" ILIKE $%d", paramCount))
			args = append(args, "%"+f.Value+"%")
			paramCount++
		default:
			return "", nil, fmt.Errorf("invalid operator: %s", f.Operator)
		}
	}

	if len(conditions) == 0 {
		return "", nil, nil
	}

	return strings.Join(conditions, " AND "), args, nil
}
