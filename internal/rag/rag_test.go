package rag

import (
	"testing"
)

func TestHashContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "empty string",
			content: "",
			want:    "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:    "simple string",
			content: "hello world",
			want:    "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		},
		{
			name:    "same content produces same hash",
			content: "test content",
			want:    HashContent("test content"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HashContent(tt.content)
			if got != tt.want {
				t.Errorf("HashContent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHashContentConsistency(t *testing.T) {
	content := "This is a test document for RAG."
	hash1 := HashContent(content)
	hash2 := HashContent(content)

	if hash1 != hash2 {
		t.Errorf("HashContent should produce consistent results: got %v and %v", hash1, hash2)
	}
}

func TestHashContentUniqueness(t *testing.T) {
	content1 := "Document one"
	content2 := "Document two"

	hash1 := HashContent(content1)
	hash2 := HashContent(content2)

	if hash1 == hash2 {
		t.Error("HashContent should produce different hashes for different content")
	}
}

func TestFloat32SliceToVector(t *testing.T) {
	tests := []struct {
		name string
		v    []float32
		want string
	}{
		{
			name: "empty slice",
			v:    []float32{},
			want: "[]",
		},
		{
			name: "single element",
			v:    []float32{1.0},
			want: "[1.000000]",
		},
		{
			name: "multiple elements",
			v:    []float32{1.0, 2.5, 3.14159},
			want: "[1.000000,2.500000,3.141590]",
		},
		{
			name: "negative values",
			v:    []float32{-1.0, 0.0, 1.0},
			want: "[-1.000000,0.000000,1.000000]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Float32SliceToVector(tt.v)
			if got != tt.want {
				t.Errorf("Float32SliceToVector() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFloat32SliceToVectorFormat(t *testing.T) {
	// Test that the format is valid for pgvector
	v := []float32{0.1, 0.2, 0.3}
	result := Float32SliceToVector(v)

	// Check format: starts with [ and ends with ]
	if result[0] != '[' || result[len(result)-1] != ']' {
		t.Errorf("Vector format should be enclosed in brackets: %v", result)
	}

	// Should contain comma separators
	if len(v) > 1 {
		commaCount := 0
		for _, c := range result {
			if c == ',' {
				commaCount++
			}
		}
		if commaCount != len(v)-1 {
			t.Errorf("Expected %d commas, got %d in: %v", len(v)-1, commaCount, result)
		}
	}
}
