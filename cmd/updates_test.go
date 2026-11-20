package main

import (
	"fmt"
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

func TestIsNewRelease(t *testing.T) {
	tests := []struct {
		name           string
		curVersion     string
		releaseVersion string
		want           bool
	}{
		{"rc to matching stable", "v2.9.0-rc.10", "v2.9.0", true},
		{"rc to older major", "v2.9.10-rc.10", "v1.99.99", false},
		{"rc to older minor", "v2.9.10-rc.10", "v2.8.99", false},
		{"rc to older patch", "v2.9.10-rc.10", "v2.9.9", false},
		{"rc to matching stable double digit patch", "v2.9.10-rc.10", "v2.9.10", true},
		{"rc to newer patch", "v2.9.10-rc.10", "v2.9.11", true},
		{"rc to newer minor", "v2.9.10-rc.10", "v2.10.0", true},
		{"rc to newer major", "v2.9.10-rc.10", "v3.0.0", true},
		{"stable to matching stable", "v2.9.0", "v2.9.0", false},
		{"stable to older major", "v2.9.10", "v1.99.99", false},
		{"stable to older minor", "v2.9.10", "v2.8.99", false},
		{"stable to older patch", "v2.9.10", "v2.9.9", false},
		{"stable to matching stable double digit patch", "v2.9.10", "v2.9.10", false},
		{"stable to newer patch", "v2.9.10", "v2.9.11", true},
		{"stable to newer minor", "v2.9.10", "v2.10.0", true},
		{"stable to newer major", "v2.9.10", "v3.0.0", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isNewRelease(tc.curVersion, tc.releaseVersion); got != tc.want {
				t.Fatalf("isNewRelease(%q, %q) = %v, want %v", tc.curVersion, tc.releaseVersion, got, tc.want)
			}
		})
	}
}

func TestIsNewReleaseCombinations(t *testing.T) {
	type version struct {
		major int
		minor int
		patch int
	}
	var versions []version
	for _, major := range []int{0, 1, 2, 9, 10} {
		for _, minor := range []int{0, 1, 2, 9, 10} {
			for _, patch := range []int{0, 1, 2, 9, 10} {
				versions = append(versions, version{major, minor, patch})
			}
		}
	}

	for _, suffix := range []string{"", "-rc.1", "-rc.9", "-rc.10"} {
		for _, current := range versions {
			curVersion := fmt.Sprintf("v%d.%d.%d%s", current.major, current.minor, current.patch, suffix)
			for _, release := range versions {
				releaseVersion := fmt.Sprintf("v%d.%d.%d", release.major, release.minor, release.patch)
				want := release.major > current.major ||
					(release.major == current.major && release.minor > current.minor) ||
					(release.major == current.major && release.minor == current.minor && release.patch > current.patch) ||
					(release == current && suffix != "")
				if got := isNewRelease(curVersion, releaseVersion); got != want {
					t.Fatalf("isNewRelease(%q, %q) = %v, want %v", curVersion, releaseVersion, got, want)
				}
			}
		}
	}
}
