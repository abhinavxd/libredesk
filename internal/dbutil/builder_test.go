package dbutil

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

var testAllowed = AllowedFields{
	"conversations": {"status_id", "priority_id", "created_at"},
	"users":         {"email"},
}

var testRenderers = FieldRenderers{
	"conversations": {
		"tags": func(operator, value string, paramIndex int) (string, []any, error) {
			switch operator {
			case "contains":
				return fmt.Sprintf("conversations.id IN (SELECT conversation_id FROM conversation_tags WHERE tag_id = ANY($%d::int[]))", paramIndex), []any{value}, nil
			case "set":
				return "EXISTS (SELECT 1 FROM conversation_tags WHERE conversation_id = conversations.id)", nil, nil
			default:
				return "", nil, fmt.Errorf("bad tag op")
			}
		},
	},
}

func build(t *testing.T, filtersJSON string) (string, []any, error) {
	t.Helper()
	return BuildPaginatedQuery("SELECT 1 FROM conversations WHERE 1=1", nil, PaginationOptions{Page: 1, PageSize: 30}, filtersJSON, testAllowed, testRenderers)
}

func TestLegacyFlatArrayIsAnded(t *testing.T) {
	q, args, err := build(t, `[{"model":"conversations","field":"status_id","operator":"equals","value":"1"},{"model":"conversations","field":"priority_id","operator":"equals","value":"2"}]`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(q, "(conversations.status_id = $1 AND conversations.priority_id = $2)") {
		t.Fatalf("expected AND join, got: %s", q)
	}
	if len(args) != 4 { // 2 filters + LIMIT + OFFSET
		t.Fatalf("expected 4 args, got %d: %v", len(args), args)
	}
}

func TestGroupOr(t *testing.T) {
	q, _, err := build(t, `{"logic":"OR","rules":[{"model":"conversations","field":"status_id","operator":"equals","value":"1"},{"model":"conversations","field":"status_id","operator":"equals","value":"5"}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(q, "(conversations.status_id = $1 OR conversations.status_id = $2)") {
		t.Fatalf("expected OR join, got: %s", q)
	}
}

func TestNestedMixed(t *testing.T) {
	q, _, err := build(t, `{"logic":"AND","rules":[{"model":"conversations","field":"priority_id","operator":"equals","value":"3"},{"logic":"OR","rules":[{"model":"conversations","field":"status_id","operator":"equals","value":"1"},{"model":"conversations","field":"status_id","operator":"equals","value":"5"}]}]}`)
	if err != nil {
		t.Fatal(err)
	}
	want := "(conversations.priority_id = $1 AND (conversations.status_id = $2 OR conversations.status_id = $3))"
	if !strings.Contains(q, want) {
		t.Fatalf("expected nested mixed clause %q, got: %s", want, q)
	}
}

func TestTagLeafInsideOrBranch(t *testing.T) {
	q, _, err := build(t, `{"logic":"OR","rules":[{"model":"conversations","field":"status_id","operator":"equals","value":"1"},{"model":"conversations","field":"tags","operator":"contains","value":"[1,2]"}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(q, "OR conversations.id IN (SELECT conversation_id FROM conversation_tags") {
		t.Fatalf("expected tag subquery inside OR, got: %s", q)
	}
}

func TestDepthTooDeepRejected(t *testing.T) {
	_, _, err := build(t, `{"logic":"AND","rules":[{"logic":"OR","rules":[{"logic":"AND","rules":[{"model":"conversations","field":"status_id","operator":"equals","value":"1"}]}]}]}`)
	if err == nil {
		t.Fatal("expected depth error")
	}
}

func TestInvalidLogicRejected(t *testing.T) {
	_, _, err := build(t, `{"logic":"XOR","rules":[{"model":"conversations","field":"status_id","operator":"equals","value":"1"}]}`)
	if err == nil {
		t.Fatal("expected invalid logic error")
	}
}

func TestInvalidFieldRejected(t *testing.T) {
	_, _, err := build(t, `[{"model":"conversations","field":"secret","operator":"equals","value":"1"}]`)
	if err == nil {
		t.Fatal("expected invalid field error")
	}
}

func TestEmptyFiltersNoClause(t *testing.T) {
	q, _, err := build(t, `[]`)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(q, "WHERE 1=1 AND") {
		t.Fatalf("expected no filter clause, got: %s", q)
	}
}

func TestContainsOnPlainColumnIsILike(t *testing.T) {
	q, args, err := build(t, `[{"model":"users","field":"email","operator":"contains","value":"gmail"}]`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(q, "users.email ILIKE $1") {
		t.Fatalf("expected ILIKE for contains, got: %s", q)
	}
	if args[0] != "%gmail%" {
		t.Fatalf("expected wrapped pattern, got: %v", args[0])
	}
	q, _, err = build(t, `[{"model":"users","field":"email","operator":"not contains","value":"gmail"}]`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(q, "users.email NOT ILIKE $1") {
		t.Fatalf("expected NOT ILIKE for not contains, got: %s", q)
	}
}

func TestTooManyGroupsRejected(t *testing.T) {
	group := `{"logic":"AND","rules":[{"model":"conversations","field":"status_id","operator":"equals","value":"1"}]}`
	groups := make([]string, MaxFilterGroups+1)
	for i := range groups {
		groups[i] = group
	}
	j := `{"logic":"OR","rules":[` + strings.Join(groups, ",") + `]}`
	_, _, err := build(t, j)
	if !errors.Is(err, ErrTooManyGroups) {
		t.Fatalf("expected ErrTooManyGroups for more than %d groups, got: %v", MaxFilterGroups, err)
	}
}

func TestTooManyConditionsRejected(t *testing.T) {
	leaf := `{"model":"conversations","field":"status_id","operator":"equals","value":"1"}`
	leaves := make([]string, maxFilterConditions+1)
	for i := range leaves {
		leaves[i] = leaf
	}
	if _, _, err := build(t, `[`+strings.Join(leaves, ",")+`]`); err == nil {
		t.Fatalf("expected error for more than %d conditions", maxFilterConditions)
	}
}

func TestTooManyInValuesRejected(t *testing.T) {
	vals := make([]string, maxInValues+1)
	for i := range vals {
		vals[i] = `"1"`
	}
	if _, _, err := build(t, `[{"model":"conversations","field":"status_id","operator":"in","value":"[`+strings.ReplaceAll(strings.Join(vals, ","), `"`, `\"`)+`]"}]`); err == nil {
		t.Fatal("expected error for oversized 'in' array")
	}
}

func TestEmptyInRejected(t *testing.T) {
	if _, _, err := build(t, `[{"model":"conversations","field":"status_id","operator":"in","value":"[]"}]`); err == nil {
		t.Fatal("expected error for empty 'in' array")
	}
}

func TestEmptyValueRejected(t *testing.T) {
	if _, _, err := build(t, `[{"model":"conversations","field":"status_id","operator":"equals","value":""}]`); err == nil {
		t.Fatal("expected error for empty value on 'equals'")
	}
	if _, _, err := build(t, `[{"model":"conversations","field":"status_id","operator":"set","value":""}]`); err != nil {
		t.Fatalf("'set' should not require a value: %v", err)
	}
}

func TestValidateFilters(t *testing.T) {
	if err := ValidateFilters(`{"logic":"AND","rules":[{"model":"conversations","field":"status_id","operator":"equals","value":"1"}]}`, testAllowed, testRenderers); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ValidateFilters(`{"logic":"AND","rules":[{"model":"conversations","field":"nope","operator":"equals","value":"1"}]}`, testAllowed, testRenderers); err == nil {
		t.Fatal("expected validation error for bad field")
	}
}
