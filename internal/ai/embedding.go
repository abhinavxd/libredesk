package ai

import (
	"context"
	"database/sql"
	"encoding/binary"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
)

// reconcileInterval is how often knowledge base content that failed to embed is retried.
const reconcileInterval = 1 * time.Minute

// indexedChunk is one embedded chunk held in memory for brute-force search.
type indexedChunk struct {
	sourceType string
	sourceID   int
	chunkText  string
	vec        []float32
	norm       float32
}

// embeddingIndex is an in-memory brute-force vector store.
type embeddingIndex struct {
	mu     sync.RWMutex
	chunks []indexedChunk
}

func newEmbeddingIndex() *embeddingIndex {
	return &embeddingIndex{}
}

func (ix *embeddingIndex) replaceAll(chunks []indexedChunk) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	ix.chunks = chunks
}

func (ix *embeddingIndex) replaceSource(sourceType string, sourceID int, chunks []indexedChunk) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	kept := ix.chunks[:0:0]
	for _, c := range ix.chunks {
		if c.sourceType == sourceType && c.sourceID == sourceID {
			continue
		}
		kept = append(kept, c)
	}
	ix.chunks = append(kept, chunks...)
}

func (ix *embeddingIndex) removeSource(sourceType string, sourceID int) {
	ix.replaceSource(sourceType, sourceID, nil)
}

// search returns the top-k matches and the count of chunks skipped for mismatched vector dimensions.
func (ix *embeddingIndex) search(query []float32, k int) ([]models.SearchResult, int) {
	ix.mu.RLock()
	defer ix.mu.RUnlock()

	qNorm := norm(query)
	if qNorm == 0 {
		return nil, 0
	}

	dimMismatch := 0
	results := make([]models.SearchResult, 0, len(ix.chunks))
	for _, c := range ix.chunks {
		if len(c.vec) != len(query) {
			dimMismatch++
			continue
		}
		if c.norm == 0 {
			continue
		}
		score := dot(query, c.vec) / (qNorm * c.norm)
		results = append(results, models.SearchResult{
			SourceType: c.sourceType,
			SourceID:   c.sourceID,
			ChunkText:  c.chunkText,
			Score:      float64(score),
		})
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	if k > 0 && len(results) > k {
		results = results[:k]
	}
	return results, dimMismatch
}

// Search embeds the query and returns the top-k most similar chunks across the whole index.
func (m *Manager) Search(ctx context.Context, query string, k int) ([]models.SearchResult, error) {
	qvec, err := m.GetEmbeddings(ctx, query)
	if err != nil {
		return nil, err
	}
	results, dimMismatch := m.index.search(qvec, k)
	if dimMismatch > 0 {
		m.lo.Warn("skipped stale embeddings with mismatched dimensions; reindex the knowledge base after changing the embedding model", "count", dimMismatch, "query_dimensions", len(qvec))
	}
	m.lo.Debug("rag search", "query_len", len(query), "hits", len(results))
	for i, r := range results {
		m.lo.Debug("rag fetched chunk", "rank", i+1, "score", r.Score, "source_type", r.SourceType, "source_id", r.SourceID, "chunk_len", len(r.ChunkText))
	}
	return results, nil
}

// Reindex re-chunks and re-embeds a source's content, replacing its stored and in-memory vectors.
func (m *Manager) Reindex(sourceType string, sourceID int, title, htmlContent string) error {
	chunks, err := stringutil.ChunkHTMLContent(title, htmlContent, m.chunkCfg)
	if err != nil {
		m.lo.Error("error chunking content for embedding", "error", err, "source_type", sourceType, "source_id", sourceID)
		return err
	}

	// Generate embeddings before opening the transaction; provider calls are slow.
	vecs, err := m.GetEmbeddingsBatch(context.Background(), chunks)
	if err != nil {
		m.lo.Error("error generating embeddings", "error", err)
		return err
	}
	indexed := make([]indexedChunk, 0, len(chunks))
	for i, chunk := range chunks {
		indexed = append(indexed, indexedChunk{
			sourceType: sourceType,
			sourceID:   sourceID,
			chunkText:  chunk,
			vec:        vecs[i],
			norm:       norm(vecs[i]),
		})
	}

	m.reindexMu.Lock()
	defer m.reindexMu.Unlock()

	tx, err := m.db.BeginTxx(context.Background(), &sql.TxOptions{})
	if err != nil {
		m.lo.Error("error beginning reindex transaction", "error", err)
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Stmtx(m.q.DeleteEmbeddingsBySource).Exec(sourceType, sourceID); err != nil {
		m.lo.Error("error clearing old embeddings", "error", err)
		return err
	}
	insert := tx.Stmtx(m.q.InsertEmbedding)
	for _, c := range indexed {
		if _, err := insert.Exec(sourceType, sourceID, c.chunkText, serializeEmbedding(c.vec), len(c.vec)); err != nil {
			m.lo.Error("error inserting embedding", "error", err)
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing reindex transaction", "error", err)
		return err
	}

	m.index.replaceSource(sourceType, sourceID, indexed)
	return nil
}

// RemoveEmbeddings drops all vectors for a source from the DB and memory.
func (m *Manager) RemoveEmbeddings(sourceType string, sourceID int) error {
	m.reindexMu.Lock()
	defer m.reindexMu.Unlock()

	if _, err := m.q.DeleteEmbeddingsBySource.Exec(sourceType, sourceID); err != nil {
		m.lo.Error("error deleting embeddings", "error", err)
		return err
	}
	m.index.removeSource(sourceType, sourceID)
	return nil
}

// loadIndex loads all stored embeddings into memory at boot.
func (m *Manager) loadIndex() error {
	m.reindexMu.Lock()
	defer m.reindexMu.Unlock()

	var rows []models.Embedding
	if err := m.q.GetAllEmbeddings.Select(&rows); err != nil {
		return err
	}
	chunks := make([]indexedChunk, 0, len(rows))
	var vectorBytes, textBytes int
	for _, r := range rows {
		vec := deserializeEmbedding(r.Embedding)
		if len(vec) == 0 {
			continue
		}
		vectorBytes += len(vec) * 4
		textBytes += len(r.ChunkText)
		chunks = append(chunks, indexedChunk{
			sourceType: r.SourceType,
			sourceID:   int(r.SourceID),
			chunkText:  r.ChunkText,
			vec:        vec,
			norm:       norm(vec),
		})
	}
	m.index.replaceAll(chunks)
	dimensions := 0
	if len(chunks) > 0 {
		dimensions = len(chunks[0].vec)
	}
	m.lo.Info("loaded embeddings into memory", "count", len(chunks), "dimensions", dimensions,
		"vector_bytes", vectorBytes, "text_bytes", textBytes,
		"approx_mb", math.Round(float64(vectorBytes+textBytes)/(1024*1024)*100)/100)
	return nil
}

// Run periodically reconciles knowledge base embeddings so content that failed to embed, or predates
// a model change, is retried without a manual re-save.
func (m *Manager) Run(ctx context.Context) {
	ticker := time.NewTicker(reconcileInterval)
	defer ticker.Stop()
	m.reconcile()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.reconcile()
		}
	}
}

// reconcile re-embeds every enabled snippet whose stored fingerprint no longer matches its current
// content and the active embedding model. A skipped run (already in progress) is retried on the next tick.
func (m *Manager) reconcile() {
	if !m.reconcileMu.TryLock() {
		return
	}
	defer m.reconcileMu.Unlock()

	cfg, err := m.getRawProviderConfig(models.ProviderTypeEmbedding)
	if err != nil {
		return
	}
	// The API key is stored encrypted, so a non-empty value is enough to know a provider is configured.
	if cfg.APIKey == "" {
		return
	}
	items, err := m.GetKnowledgeBaseItems()
	if err != nil {
		return
	}

	reindexed := 0
	for _, item := range items {
		if !item.Enabled {
			// A disabled item should carry no embeddings; clean up any left behind.
			if item.EmbeddedFingerprint != "" {
				m.reindexSnippetWith(item, cfg.Model, cfg.Dimensions)
			}
			continue
		}
		if item.EmbeddedFingerprint == snippetFingerprint(item, cfg.Model, cfg.Dimensions) {
			continue
		}
		m.reindexSnippetWith(item, cfg.Model, cfg.Dimensions)
		reindexed++
	}
	if reindexed > 0 {
		m.lo.Info("reconciled knowledge base embeddings", "reindexed", reindexed)
	}
}

func serializeEmbedding(vec []float32) []byte {
	buf := make([]byte, 4*len(vec))
	for i, f := range vec {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

func deserializeEmbedding(b []byte) []float32 {
	n := len(b) / 4
	vec := make([]float32, n)
	for i := range n {
		vec[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return vec
}

func dot(a, b []float32) float32 {
	var sum float32
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}

func norm(a []float32) float32 {
	var sum float32
	for _, v := range a {
		sum += v * v
	}
	return float32(math.Sqrt(float64(sum)))
}
