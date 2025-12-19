package models

import (
	"encoding/json"
	"time"
)

// Source represents a knowledge source configuration.
type Source struct {
	ID           int             `db:"id" json:"id"`
	CreatedAt    time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at" json:"updated_at"`
	Name         string          `db:"name" json:"name"`
	SourceType   string          `db:"source_type" json:"source_type"`
	Config       json.RawMessage `db:"config" json:"config"`
	Enabled      bool            `db:"enabled" json:"enabled"`
	LastSyncedAt *time.Time      `db:"last_synced_at" json:"last_synced_at"`
}

// Document represents a chunk of knowledge with its embedding.
type Document struct {
	ID          int             `db:"id" json:"id"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updated_at"`
	SourceID    int             `db:"source_id" json:"source_id"`
	SourceRef   string          `db:"source_ref" json:"source_ref"`
	Title       string          `db:"title" json:"title"`
	Content     string          `db:"content" json:"content"`
	ContentHash string          `db:"content_hash" json:"content_hash"`
	Metadata    json.RawMessage `db:"metadata" json:"metadata"`
}

// SearchResult represents a document with similarity score.
type SearchResult struct {
	Document
	Similarity float64 `db:"similarity" json:"similarity"`
}

// WebpageConfig holds configuration for webpage sources.
type WebpageConfig struct {
	URLs []string `json:"urls"`
}

// MacroConfig holds configuration for macro sources.
type MacroConfig struct {
	IncludeAll bool  `json:"include_all"`
	MacroIDs   []int `json:"macro_ids,omitempty"`
}

// FileConfig holds configuration for file-based sources.
type FileConfig struct {
	Filename string `json:"filename"`
	FileType string `json:"file_type"` // txt, csv, json
	Content  string `json:"content"`   // The actual file content
}
