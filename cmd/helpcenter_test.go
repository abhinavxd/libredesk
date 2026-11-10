package main

import (
	"slices"
	"testing"

	hcmodels "github.com/abhinavxd/libredesk/internal/helpcenter/models"
)

func TestCollectionTrail(t *testing.T) {
	tree := []hcmodels.TreeCollection{
		{Collection: hcmodels.Collection{Slug: "other"}},
		{
			Collection: hcmodels.Collection{Slug: "root"},
			Children: []hcmodels.TreeCollection{{
				Collection: hcmodels.Collection{Slug: "child"},
				Children:   []hcmodels.TreeCollection{{Collection: hcmodels.Collection{Slug: "leaf"}}},
			}},
		},
	}
	for _, tt := range []struct {
		slug string
		want []string
	}{
		{"root", []string{"root"}},
		{"child", []string{"root", "child"}},
		{"leaf", []string{"root", "child", "leaf"}},
		{"other", []string{"other"}},
		{"missing", nil},
		{"", nil},
	} {
		t.Run(tt.slug, func(t *testing.T) {
			var got []string
			for _, c := range collectionTrail(tree, tt.slug) {
				got = append(got, c.Slug)
			}
			if !slices.Equal(got, tt.want) {
				t.Fatalf("collectionTrail(%q) = %v, want %v", tt.slug, got, tt.want)
			}
		})
	}
}
