package resourceimage

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/abhinavxd/libredesk/internal/resourcepolicy"
	"github.com/jmoiron/sqlx"
	"golang.org/x/sync/errgroup"
)

func SourceID(source string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(source)))
}

func Sources(content string) map[string]string {
	policy, _ := resourcepolicy.New(resourcepolicy.Config{Mode: resourcepolicy.LoadOnReceipt})
	sources := make(map[string]string)
	policy.PrepareDisplayWithImages(content, nil, func(source string) string {
		if len(sources) < 32 {
			sources[SourceID(source)] = source
		}
		return ""
	})
	return sources
}

func (s *Store) Prefetch(ctx context.Context, messageID int, content string, readPolicy func(*sqlx.Tx) (resourcepolicy.Config, error)) error {
	cfg, err := readPolicy(nil)
	if err != nil {
		return err
	}
	if cfg.Mode != resourcepolicy.LoadOnReceipt {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	var group errgroup.Group
	group.SetLimit(4)
	for id, source := range Sources(content) {
		if err := ctx.Err(); err != nil {
			group.Wait()
			return err
		}
		group.Go(func() error {
			_, err := s.get(ctx, mmodels.ModelResourceImages, messageID, id, source, func(tx *sqlx.Tx) error {
				cfg, err := readPolicy(tx)
				if err != nil {
					return err
				}
				policy, err := resourcepolicy.New(cfg)
				if err != nil {
					return err
				}
				if cfg.Mode != resourcepolicy.LoadOnReceipt || !policy.AllowsURL(source) {
					return ErrImage
				}
				return nil
			})
			return err
		})
	}
	return group.Wait()
}
