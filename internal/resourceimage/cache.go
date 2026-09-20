package resourceimage

import (
	"context"
	"fmt"
	"time"

	"github.com/abhinavxd/libredesk/internal/resourcepolicy"
	"github.com/jmoiron/sqlx"
)

// Uploads normally have a 30-second request deadline. Retain their reservations
// longer so an interrupted process can be cleaned up without holding a DB lock
// during storage I/O. Expired reservations cannot publish a cache entry.
const reservationLifetime = 5 * time.Minute

type cacheUsage struct {
	Bytes    int64 `db:"bytes"`
	Deleting int64 `db:"deleting"`
}

const cacheUsageSQL = `SELECT
 COALESCE((SELECT SUM(size) FROM media WHERE model_type IN ('resource_images', 'resource_avatars')), 0)
 + COALESCE(SUM(size), 0) AS bytes,
 COALESCE(SUM(size) FILTER (WHERE state = 'delete'), 0) AS deleting
 FROM resource_image_cache_pending`

// This lock protects only short accounting transactions, never network I/O.
func lockCache(ctx context.Context, tx *sqlx.Tx) error {
	_, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('libredesk:resource-image-cache', 0))`)
	return err
}

func (s *Store) cacheLimit(tx *sqlx.Tx) (int64, error) {
	cfg, err := s.readPolicy(tx)
	if err != nil {
		return 0, err
	}
	cfg, err = resourcepolicy.Normalize(cfg)
	return cfg.MaxCacheBytes, err
}

func (s *Store) reserve(ctx context.Context, name string, size int64) error {
	if size <= 0 {
		return ErrImage
	}
	for {
		ready, err := s.makeSpace(ctx, name, size)
		if err != nil || ready {
			return err
		}
		deleted, err := s.deletePending(ctx)
		if err != nil {
			return err
		}
		if deleted == 0 {
			if err := waitForCache(ctx); err != nil {
				return err
			}
		}
	}
}

// makeSpace reserves capacity or moves FIFO victims into the deletion queue.
// Queued files still count against the budget until storage confirms deletion.
func (s *Store) makeSpace(ctx context.Context, name string, size int64) (bool, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	if err := lockCache(ctx, tx); err != nil {
		return false, err
	}
	limit, err := s.cacheLimit(tx)
	if err != nil {
		return false, err
	}
	// Reject before evicting anything for an image that can never fit.
	if size > limit {
		return false, ErrImage
	}
	if _, err := tx.ExecContext(ctx, `UPDATE resource_image_cache_pending SET state = 'delete'
 WHERE state = 'upload' AND created_at < $1`, time.Now().Add(-reservationLifetime)); err != nil {
		return false, err
	}
	var usage cacheUsage
	if err := tx.GetContext(ctx, &usage, cacheUsageSQL); err != nil {
		return false, err
	}
	if usage.Bytes+size <= limit {
		if name != "" {
			if _, err := tx.ExecContext(ctx, `INSERT INTO resource_image_cache_pending (uuid, size, state) VALUES ($1, $2, 'upload')`, name, size); err != nil {
				return false, err
			}
		}
		return true, tx.Commit()
	}

	// Other requests may already be deleting enough bytes. Do not evict more
	// just because those storage operations have not finished yet.
	needed := usage.Bytes + size - limit - usage.Deleting
	if needed > 0 {
		var victims []struct {
			UUID string `db:"uuid"`
			Size int64  `db:"size"`
		}
		if err := tx.SelectContext(ctx, &victims, `SELECT uuid, size FROM media
 WHERE model_type IN ('resource_images', 'resource_avatars')
 ORDER BY created_at, id LIMIT 128 FOR UPDATE`); err != nil {
			return false, err
		}
		for _, victim := range victims {
			if _, err := tx.ExecContext(ctx, `WITH removed AS (
 DELETE FROM media WHERE uuid = $1 RETURNING uuid, size
) INSERT INTO resource_image_cache_pending (uuid, size, state)
 SELECT uuid, size, 'delete' FROM removed`, victim.UUID); err != nil {
				return false, err
			}
			needed -= victim.Size
			if needed <= 0 {
				break
			}
		}
	}
	return false, tx.Commit()
}

func (s *Store) validateReservation(ctx context.Context, tx *sqlx.Tx, name string) error {
	limit, err := s.cacheLimit(tx)
	if err != nil {
		return err
	}
	var valid bool
	if err := tx.GetContext(ctx, &valid, `SELECT EXISTS (SELECT 1 FROM resource_image_cache_pending
 WHERE uuid = $1 AND state = 'upload' AND created_at >= $2)`, name, time.Now().Add(-reservationLifetime)); err != nil {
		return err
	}
	var usage cacheUsage
	if err := tx.GetContext(ctx, &usage, cacheUsageSQL); err != nil {
		return err
	}
	if !valid || usage.Bytes > limit {
		return ErrImage
	}
	return nil
}

func (s *Store) deletePending(ctx context.Context) (int, error) {
	var names []string
	if err := s.db.SelectContext(ctx, &names, `SELECT uuid FROM resource_image_cache_pending
 WHERE state = 'delete' ORDER BY created_at, uuid LIMIT 128`); err != nil {
		return 0, err
	}
	for i, name := range names {
		if err := ctx.Err(); err != nil {
			return i, err
		}
		if err := s.media.Delete(name); err != nil {
			return i, fmt.Errorf("delete cached image %s: %w", name, err)
		}
		if _, err := s.db.ExecContext(ctx, `DELETE FROM resource_image_cache_pending WHERE uuid = $1 AND state = 'delete'`, name); err != nil {
			return i, err
		}
	}
	return len(names), nil
}

func (s *Store) discardUpload(name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Leave failed deletes queued and accounted for so the cleaner can retry.
	if _, err := s.db.ExecContext(ctx, `UPDATE resource_image_cache_pending SET state = 'delete' WHERE uuid = $1`, name); err == nil {
		if err := s.media.Delete(name); err == nil {
			s.db.ExecContext(ctx, `DELETE FROM resource_image_cache_pending WHERE uuid = $1 AND state = 'delete'`, name)
		}
	}
	s.RequestCleanup()
}

func waitForCache(ctx context.Context) error {
	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Trim applies the current limit without fetching any images.
func (s *Store) Trim(ctx context.Context) error {
	for {
		ready, err := s.makeSpace(ctx, "", 0)
		if err != nil {
			return err
		}
		deleted, err := s.deletePending(ctx)
		if err != nil {
			return err
		}
		if ready && deleted == 0 {
			return nil
		}
		if !ready && deleted == 0 {
			if err := waitForCache(ctx); err != nil {
				return err
			}
		}
	}
}

func (s *Store) RequestCleanup() {
	select {
	case s.cleanup <- struct{}{}:
	default:
	}
}

// RunCacheCleaner starts with a sweep, wakes on settings changes, and retries
// interrupted uploads or failed storage deletes periodically.
func (s *Store) RunCacheCleaner(ctx context.Context, reportError func(error)) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		cleanupCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		err := s.Trim(cleanupCtx)
		cancel()
		if err != nil && ctx.Err() == nil {
			reportError(err)
		}
		select {
		case <-ctx.Done():
			return
		case <-s.cleanup:
		case <-ticker.C:
		}
	}
}
