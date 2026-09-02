package main

import (
	"fmt"
	"testing"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func requestWithQuery(query string) *fastglue.Request {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/api/v1/thing?" + query)
	return &fastglue.Request{RequestCtx: ctx}
}

func TestGetIDsParam(t *testing.T) {
	tests := []struct {
		query string
		want  []int
	}{
		{"", nil},
		{"ids=", nil},
		{"ids=%20%20", nil},
		{"ids=5", []int{5}},
		{"ids=5,7,9", []int{5, 7, 9}},
		{"ids=%205%20,%207%20", []int{5, 7}},
		{"ids=5,5,5", []int{5, 5, 5}},
		{"ids=abc", nil},
		{"ids=0", nil},
		{"ids=-3", nil},
		{"ids=1,abc,2", []int{1, 2}},
		{"ids=1,,2", []int{1, 2}},
		{"ids=1,0,-2,3", []int{1, 3}},
		{"ids=9223372036854775808", nil},
		{"other=1", nil},
	}

	for _, tc := range tests {
		got := getIDsParam(requestWithQuery(tc.query), "ids")
		if len(got) != len(tc.want) {
			t.Fatalf("query %q = %v, want %v", tc.query, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("query %q = %v, want %v", tc.query, got, tc.want)
			}
		}
	}
}

func TestGetIDsParamCap(t *testing.T) {
	query := "ids=1"
	for i := 2; i <= maxIDsParam+50; i++ {
		query += fmt.Sprintf(",%d", i)
	}

	got := getIDsParam(requestWithQuery(query), "ids")

	if len(got) != maxIDsParam {
		t.Fatalf("got %d ids, want the cap of %d", len(got), maxIDsParam)
	}
	if got[0] != 1 || got[maxIDsParam-1] != maxIDsParam {
		t.Fatalf("cap dropped the wrong end: first %d, last %d", got[0], got[maxIDsParam-1])
	}
}

func TestGetPaginationDefaultsAndClamps(t *testing.T) {
	tests := []struct {
		query        string
		wantPage     int
		wantPageSize int
	}{
		{"", 1, 30},
		{"page=0&page_size=0", 1, 30},
		{"page=-4&page_size=-9", 1, 30},
		{"page=abc&page_size=abc", 1, 30},
		{"page=3&page_size=50", 3, 50},
		{"page_size=100000", 1, maxPageSize},
	}

	for _, tc := range tests {
		page, pageSize := getPagination(requestWithQuery(tc.query))
		if page != tc.wantPage || pageSize != tc.wantPageSize {
			t.Fatalf("query %q = page %d size %d, want page %d size %d", tc.query, page, pageSize, tc.wantPage, tc.wantPageSize)
		}
	}
}

func TestGetOptionalPaginationFetchesEverythingWithoutParams(t *testing.T) {
	tests := []struct {
		query        string
		wantPage     int
		wantPageSize int
	}{
		{"", 1, 0},
		{"page=2", 2, 0},
		{"page_size=0", 1, 0},
		{"page_size=-5", 1, 0},
		{"page=2&page_size=50", 2, 50},
		{"page_size=100000", 1, maxPageSize},
	}

	for _, tc := range tests {
		page, pageSize := getOptionalPagination(requestWithQuery(tc.query))
		if page != tc.wantPage || pageSize != tc.wantPageSize {
			t.Fatalf("query %q = page %d size %d, want page %d size %d", tc.query, page, pageSize, tc.wantPage, tc.wantPageSize)
		}
	}
}

func TestPageOffset(t *testing.T) {
	tests := []struct {
		page     int
		pageSize int
		want     int
	}{
		{1, 30, 0},
		{2, 30, 30},
		{3, 50, 100},
		{1, 0, 0},
		{5, 0, 0},
	}

	for _, tc := range tests {
		if got := dbutil.PageOffset(tc.page, tc.pageSize); got != tc.want {
			t.Fatalf("PageOffset(%d, %d) = %d, want %d", tc.page, tc.pageSize, got, tc.want)
		}
	}
}
