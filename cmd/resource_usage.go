package main

import (
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/media"
	"github.com/abhinavxd/libredesk/internal/resourceusage"
	"github.com/zerodha/fastglue"
)

type resourceUsageResponse struct {
	SampledAt  time.Time                 `json:"sampled_at"`
	Memory     resourceusage.MemoryUsage `json:"memory"`
	Disk       resourceusage.DiskUsage   `json:"disk"`
	Storage    media.StorageUsage        `json:"storage"`
	ImageCache imageCacheUsage           `json:"image_cache"`
}

type imageCacheUsage struct {
	UsedBytes    int64    `json:"used_bytes"`
	LimitBytes   int64    `json:"limit_bytes"`
	UsagePercent *float64 `json:"usage_percent"`
	LimitSource  string   `json:"limit_source"`
}

// handleGetResourceUsage returns container-local resource metrics for admins.
// Disk metrics describe the filesystem containing the upload path; they do not
// represent the host, Docker's writable layer, or the Postgres volume.
func handleGetResourceUsage(r *fastglue.Request) error {
	app := r.Context.(*App)
	snapshot := resourceusage.Collect(ko.String("upload.fs.upload_path"))
	storage, err := app.media.GetStorageUsage()
	if err != nil {
		app.lo.Error("error reading durable media usage", "error", err)
		return r.SendErrorEnvelope(500, "Unable to read resource usage", nil, envelope.GeneralError)
	}
	imageCacheBytes, err := app.resourceImages.Usage()
	if err != nil {
		app.lo.Error("error reading external image cache usage", "error", err)
		return r.SendErrorEnvelope(500, "Unable to read resource usage", nil, envelope.GeneralError)
	}
	policy, err := app.setting.GetResourcePolicy()
	if err != nil {
		app.lo.Error("error reading external image cache policy", "error", err)
		return r.SendErrorEnvelope(500, "Unable to read resource usage", nil, envelope.GeneralError)
	}
	imagePercent := float64(imageCacheBytes) * 100 / float64(policy.MaxCacheBytes)
	return r.SendEnvelope(resourceUsageResponse{
		SampledAt: snapshot.SampledAt,
		Memory:    snapshot.Memory,
		Disk:      snapshot.Disk,
		Storage:   storage,
		ImageCache: imageCacheUsage{
			UsedBytes:    imageCacheBytes,
			LimitBytes:   policy.MaxCacheBytes,
			UsagePercent: &imagePercent,
			LimitSource:  "image_cache_policy",
		},
	})
}
