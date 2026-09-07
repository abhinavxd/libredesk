package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const updatesPayload = `{
    "update": {
        "release_version": "v2.7.1",
        "release_date": "2026-08-12",
        "url": "https://github.com/abhinavxd/libredesk/releases/tag/v2.7.1",
        "description": "Bug fixes and improvements."
    },
    "messages": []
}`

func TestFetchAppUpdateSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(updatesPayload))
	}))
	defer srv.Close()

	out, err := fetchAppUpdate(srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Update.ReleaseVersion != "v2.7.1" {
		t.Fatalf("got release version %q, want v2.7.1", out.Update.ReleaseVersion)
	}
	if out.Update.ReleaseDate != "2026-08-12" {
		t.Fatalf("got release date %q, want 2026-08-12", out.Update.ReleaseDate)
	}
	if out.Update.URL != "https://github.com/abhinavxd/libredesk/releases/tag/v2.7.1" {
		t.Fatalf("got url %q", out.Update.URL)
	}
	if out.Update.Description != "Bug fixes and improvements." {
		t.Fatalf("got description %q", out.Update.Description)
	}
	if len(out.Messages) != 0 {
		t.Fatalf("got %d messages, want 0", len(out.Messages))
	}
}

func TestFetchAppUpdateNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	out, err := fetchAppUpdate(srv.Client(), srv.URL)
	if err == nil {
		t.Fatal("expected an error for a non-200 status")
	}
	if out != nil {
		t.Fatalf("expected nil result on error, got %+v", out)
	}
}
