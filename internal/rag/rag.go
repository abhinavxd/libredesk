// Package rag handles Retrieval Augmented Generation for AI responses.
package rag

import (
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/rag/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// EmbeddingFunc generates embeddings for text.
type EmbeddingFunc func(text string) ([]float32, error)

// Manager handles RAG operations.
type Manager struct {
	q             queries
	db            *sqlx.DB
	lo            *logf.Logger
	i18n          *i18n.I18n
	embeddingFunc EmbeddingFunc
}

type queries struct {
	GetSources             *sqlx.Stmt `query:"get-sources"`
	GetSource              *sqlx.Stmt `query:"get-source"`
	GetEnabledSources      *sqlx.Stmt `query:"get-enabled-sources"`
	CreateSource           *sqlx.Stmt `query:"create-source"`
	UpdateSource           *sqlx.Stmt `query:"update-source"`
	DeleteSource           *sqlx.Stmt `query:"delete-source"`
	UpdateSourceSynced     *sqlx.Stmt `query:"update-source-synced"`
	GetDocumentsBySource   *sqlx.Stmt `query:"get-documents-by-source"`
	GetDocumentBySourceRef *sqlx.Stmt `query:"get-document-by-source-ref"`
	DeleteDocument         *sqlx.Stmt `query:"delete-document"`
	DeleteDocumentsBySource *sqlx.Stmt `query:"delete-documents-by-source"`
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB            *sqlx.DB
	Lo            *logf.Logger
	I18n          *i18n.I18n
	EmbeddingFunc EmbeddingFunc
}

// New creates a new RAG manager.
func New(opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		q:             q,
		db:            opts.DB,
		lo:            opts.Lo,
		i18n:          opts.I18n,
		embeddingFunc: opts.EmbeddingFunc,
	}, nil
}

// SetEmbeddingFunc sets the embedding function.
func (m *Manager) SetEmbeddingFunc(fn EmbeddingFunc) {
	m.embeddingFunc = fn
}

// GetDB returns the database connection for raw queries.
func (m *Manager) GetDB() *sqlx.DB {
	return m.db
}

// GetSources returns all knowledge sources.
func (m *Manager) GetSources() ([]models.Source, error) {
	sources := make([]models.Source, 0)
	if err := m.q.GetSources.Select(&sources); err != nil {
		m.lo.Error("error fetching sources", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "knowledge sources"), nil)
	}
	return sources, nil
}

// GetSource returns a source by ID.
func (m *Manager) GetSource(id int) (models.Source, error) {
	var source models.Source
	if err := m.q.GetSource.Get(&source, id); err != nil {
		if err == sql.ErrNoRows {
			return source, envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", "knowledge source"), nil)
		}
		m.lo.Error("error fetching source", "error", err)
		return source, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "knowledge source"), nil)
	}
	return source, nil
}

// CreateSource creates a new knowledge source.
func (m *Manager) CreateSource(name, sourceType string, config json.RawMessage, enabled bool) (models.Source, error) {
	var source models.Source
	if err := m.q.CreateSource.Get(&source, name, sourceType, config, enabled); err != nil {
		m.lo.Error("error creating source", "error", err)
		return source, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorCreating", "name", "knowledge source"), nil)
	}
	return source, nil
}

// UpdateSource updates a knowledge source.
func (m *Manager) UpdateSource(id int, name string, config json.RawMessage, enabled bool) (models.Source, error) {
	var source models.Source
	if err := m.q.UpdateSource.Get(&source, id, name, config, enabled); err != nil {
		m.lo.Error("error updating source", "error", err)
		return source, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "knowledge source"), nil)
	}
	return source, nil
}

// DeleteSource deletes a knowledge source and its documents.
func (m *Manager) DeleteSource(id int) error {
	result, err := m.q.DeleteSource.Exec(id)
	if err != nil {
		m.lo.Error("error deleting source", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "knowledge source"), nil)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", "knowledge source"), nil)
	}
	return nil
}

// UpdateSourceSynced updates the last_synced_at timestamp.
func (m *Manager) UpdateSourceSynced(id int) error {
	_, err := m.q.UpdateSourceSynced.Exec(id)
	if err != nil {
		m.lo.Error("error updating source synced time", "error", err)
	}
	return err
}

// HashContent generates a SHA256 hash of content for change detection.
func HashContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

// GetDocumentBySourceRef gets a document by source ID and source ref.
func (m *Manager) GetDocumentBySourceRef(sourceID int, sourceRef string) (models.Document, error) {
	var doc models.Document
	err := m.q.GetDocumentBySourceRef.Get(&doc, sourceID, sourceRef)
	return doc, err
}

// AddDocument adds or updates a document with its embedding.
func (m *Manager) AddDocument(sourceID int, sourceRef, title, content string, metadata json.RawMessage) error {
	if m.embeddingFunc == nil {
		return fmt.Errorf("embedding function not configured")
	}

	contentHash := HashContent(content)

	// Check if document exists and content hasn't changed
	existing, err := m.GetDocumentBySourceRef(sourceID, sourceRef)
	if err == nil && existing.ContentHash == contentHash {
		// Content unchanged, skip update
		return nil
	}

	// Generate embedding
	embedding, err := m.embeddingFunc(content)
	if err != nil {
		m.lo.Error("error generating embedding", "error", err)
		return fmt.Errorf("generating embedding: %w", err)
	}

	// Convert embedding to pgvector format
	embeddingStr := Float32SliceToVector(embedding)

	// Insert document with raw SQL for vector type
	_, err = m.db.Exec(`
		INSERT INTO rag_documents (source_id, source_ref, title, content, content_hash, embedding, metadata)
		VALUES ($1, $2, $3, $4, $5, $6::vector, $7)
		ON CONFLICT (source_id, source_ref) WHERE source_ref IS NOT NULL
		DO UPDATE SET
			title = EXCLUDED.title,
			content = EXCLUDED.content,
			content_hash = EXCLUDED.content_hash,
			embedding = EXCLUDED.embedding,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
	`, sourceID, sourceRef, title, content, contentHash, embeddingStr, metadata)

	if err != nil {
		m.lo.Error("error inserting document", "error", err)
		return fmt.Errorf("inserting document: %w", err)
	}

	return nil
}

// Search finds documents similar to the query.
func (m *Manager) Search(query string, limit int, threshold float64) ([]models.SearchResult, error) {
	if m.embeddingFunc == nil {
		m.lo.Error("embedding function not configured")
		return nil, fmt.Errorf("embedding function not configured")
	}

	m.lo.Info("RAG search started", "query", query, "limit", limit, "threshold", threshold)

	// Generate query embedding
	embedding, err := m.embeddingFunc(query)
	if err != nil {
		m.lo.Error("error generating query embedding", "error", err)
		return nil, fmt.Errorf("generating query embedding: %w", err)
	}

	m.lo.Info("RAG embedding generated", "embedding_length", len(embedding))

	embeddingStr := Float32SliceToVector(embedding)

	results := make([]models.SearchResult, 0)
	err = m.db.Select(&results, `
		SELECT
			id, created_at, updated_at, source_id, source_ref, title, content, content_hash, metadata,
			1 - (embedding <=> $1::vector) as similarity
		FROM rag_documents
		WHERE embedding IS NOT NULL
			AND 1 - (embedding <=> $1::vector) >= $3
		ORDER BY embedding <=> $1::vector
		LIMIT $2
	`, embeddingStr, limit, threshold)

	if err != nil {
		m.lo.Error("error searching documents", "error", err)
		return nil, fmt.Errorf("searching documents: %w", err)
	}

	m.lo.Info("RAG search complete", "results_count", len(results))

	return results, nil
}

// Float32SliceToVector converts a float32 slice to pgvector string format.
func Float32SliceToVector(v []float32) string {
	result := "["
	for i, f := range v {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf("%f", f)
	}
	result += "]"
	return result
}
