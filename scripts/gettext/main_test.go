package main

import "testing"

func TestTemplateExtraction(t *testing.T) {
	source := "{{/* .L.GetD \"templates\" \"Not used\" */}}\n" +
		"{{define \"body\"}}{{if .Enabled}}{{printf \"%s\" (.L.GetD \"templates\" `Search`)}}" +
		"{{else}}{{with .Locale}}{{$l := .}}{{$l.GetND \"templates\" \"%d result\" \"%d results\" 2 2}}{{end}}{{end}}{{end}}\n" +
		"{{template \"body\" .}}"
	c := catalog{}
	if err := extractTemplate("example.html", source, c); err != nil {
		t.Fatal(err)
	}
	if len(c) != 2 || c["Search"] == nil || c["%d result"] == nil {
		t.Fatalf("unexpected extracted messages: %#v", c)
	}
	if c["%d result"].plural != "%d results" {
		t.Fatal("plural was not extracted")
	}
	if !c["Search"].refs["example.html:2"] {
		t.Fatalf("incorrect source location: %v", c["Search"].refs)
	}
}

func TestTemplateExtractionRejectsInvalidCalls(t *testing.T) {
	for _, source := range []string{
		`{{ .L.Get "Search" }}`,
		`{{ .L.GetD .Domain "Search" }}`,
		`{{ .L.GetD "backend" "Search" }}`,
		`{{ .L.GetND "templates" "Result" .Plural 2 }}`,
		`{{ if }}`,
	} {
		t.Run(source, func(t *testing.T) {
			if err := extractTemplate("invalid.html", source, catalog{}); err == nil {
				t.Fatal("expected extraction error")
			}
		})
	}
}
