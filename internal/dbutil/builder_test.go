package dbutil

import (
	"errors"
	"fmt"
	"slices"
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

func buildTZ(t *testing.T, loc, filtersJSON string) (string, []any, error) {
	t.Helper()
	return BuildPaginatedQuery("SELECT 1 FROM conversations WHERE 1=1", nil, PaginationOptions{Page: 1, PageSize: 30, Location: loc}, filtersJSON, testAllowed, testRenderers)
}

func TestDateFilterResolvesInConfiguredTimezone(t *testing.T) {
	q, args, err := buildTZ(t, "Asia/Kolkata", `[{"model":"conversations","field":"created_at","operator":"equals","value":"2026-06-08"}]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(q, "AT TIME ZONE") {
		t.Fatalf("expected AT TIME ZONE in query, got: %s", q)
	}
	if !slices.Contains(args, any("2026-06-08")) || !slices.Contains(args, any("Asia/Kolkata")) {
		t.Fatalf("expected date and timezone bound as params, got args: %v", args)
	}
}

func TestDateFilterOperatorsBindDateAndTimezone(t *testing.T) {
	cases := []struct {
		op    string
		value string
	}{
		{"equals", "2026-06-08"},
		{"not equals", "2026-06-08"},
		{"greater than", "2026-06-08"},
		{"less than", "2026-06-08"},
		{"between", "2026-06-08,2026-06-10"},
	}
	for _, c := range cases {
		filter := fmt.Sprintf(`[{"model":"conversations","field":"created_at","operator":%q,"value":%q}]`, c.op, c.value)
		q, args, err := buildTZ(t, "Asia/Kolkata", filter)
		if err != nil {
			t.Fatalf("op %q: unexpected error: %v", c.op, err)
		}
		if !strings.Contains(q, "AT TIME ZONE") {
			t.Fatalf("op %q: expected AT TIME ZONE, got: %s", c.op, q)
		}
		if !slices.Contains(args, any("Asia/Kolkata")) {
			t.Fatalf("op %q: timezone not bound as a param, args: %v", c.op, args)
		}
	}
}

func TestDateFilterInvalidTimezoneFallsBackToUTC(t *testing.T) {
	for _, loc := range []string{"", "Mars/Olympus", "'; DROP TABLE users;--"} {
		_, args, err := buildTZ(t, loc, `[{"model":"conversations","field":"created_at","operator":"equals","value":"2026-06-08"}]`)
		if err != nil {
			t.Fatalf("loc %q: unexpected error: %v", loc, err)
		}
		if !slices.Contains(args, any("UTC")) {
			t.Fatalf("loc %q: expected UTC fallback in args, got: %v", loc, args)
		}
		if slices.Contains(args, any(loc)) && loc != "" {
			t.Fatalf("loc %q: invalid timezone leaked into args: %v", loc, args)
		}
	}
}
