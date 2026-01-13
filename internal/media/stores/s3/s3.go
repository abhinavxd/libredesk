// Package s3 provides an implementation of the media.Store interface for AWS S3 storage.
// It allows uploading, retrieving, and deleting files in an S3 bucket.
package s3

import (
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/media"
	"github.com/rhnvrm/simples3"
)

// Opt holds configuration parameters specific to AWS S3.
type Opt struct {
	URL        string        `koanf:"url"`
	PublicURL  string        `koanf:"public_url"`
	AccessKey  string        `koanf:"aws_access_key_id"`
	SecretKey  string        `koanf:"aws_secret_access_key"`
	Region     string        `koanf:"aws_default_region"`
	Bucket     string        `koanf:"bucket"`
	BucketPath string        `koanf:"bucket_path"`
	BucketType string        `koanf:"bucket_type"`
	UploadURI  string        `koanf:"upload_uri"`
	Expiry     time.Duration `koanf:"expiry"`
}

// Client implements the media.Store interface using AWS S3.
type Client struct {
	s3   *simples3.S3
	opts Opt
}

// New creates and initializes a new S3 client with the provided options.
// It sets up the `simples3` client for interacting with AWS S3 APIs.
func New(opt Opt) (media.Store, error) {
	var cl *simples3.S3

	if opt.URL == "" {
		opt.URL = fmt.Sprintf("https://s3.%s.amazonaws.com", opt.Region)
	}
	opt.URL = strings.TrimRight(opt.URL, "/")

	// Default expiry duration for S3 URLs.
	if opt.Expiry.Seconds() < 1 {
		opt.Expiry = 7 * 24 * time.Hour // Default to 7 days
	}

	if opt.AccessKey == "" && opt.SecretKey == "" {
		cl, _ = simples3.NewUsingIAM(opt.Region)
	} else {
		cl = simples3.New(opt.Region, opt.AccessKey, opt.SecretKey)
	}

	cl.SetEndpoint(opt.URL)

	return &Client{
		s3:   cl,
		opts: opt,
	}, nil
}

// Put uploads a file to S3 with the specified name, content type, and file content.
// It returns the name of the file or an error if the upload fails.
func (c *Client) Put(name string, cType string, file io.ReadSeeker) (string, error) {
	p := simples3.UploadInput{
		Bucket:      c.opts.Bucket,
		ContentType: cType,
		FileName:    name,
		Body:        file,
		// Paths inside the bucket should not start with /.
		ObjectKey: c.makeBucketPath(name),
	}

	if c.opts.BucketType == "public" {
		p.ACL = "public-read"
	}

	if _, err := c.s3.FilePut(p); err != nil {
		return "", err
	}

	return name, nil
}

// GetURL generates a URL to access the file stored in S3.
// It returns a pre-signed URL for private buckets or a public URL for public buckets.
func (c *Client) GetURL(name string, disposition, fileName string) string {
	if c.opts.BucketType == "private" && c.opts.PublicURL == "" {
		u := c.s3.GeneratePresignedURL(simples3.PresignedInput{
			Bucket:                     c.opts.Bucket,
			ObjectKey:                  c.makeBucketPath(name),
			Method:                     "GET",
			Timestamp:                  time.Now(),
			ExpirySeconds:              int(c.opts.Expiry.Seconds()),
			ResponseContentDisposition: fmt.Sprintf("%s; filename=\"%s\"", disposition, fileName),
		})
		return u
	}

	return c.makeFileURL(name)
}

// GetBlob retrieves the file content from S3 as a byte slice.
// It parses the URL, downloads the file, and returns its content or an error.
func (c *Client) GetBlob(uurl string) ([]byte, error) {
	if p, err := url.Parse(uurl); err != nil {
		uurl = filepath.Base(uurl)
	} else {
		uurl = filepath.Base(p.Path)
	}

	file, err := c.s3.FileDownload(simples3.DownloadInput{
		Bucket:    c.opts.Bucket,
		ObjectKey: c.makeBucketPath(filepath.Base(uurl)),
	})
	if err != nil {
		return nil, err
	}
	defer file.Close()

	b, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Delete removes the file identified by name from S3.
// It returns an error if the deletion fails.
func (c *Client) Delete(name string) error {
	err := c.s3.FileDelete(simples3.DeleteInput{
		Bucket:    c.opts.Bucket,
		ObjectKey: c.makeBucketPath(name),
	})
	return err
}

// makeBucketPath constructs the path for the file inside the bucket.
// It ensures the path does not start with a slash.
func (c *Client) makeBucketPath(name string) string {
	p := strings.TrimPrefix(strings.TrimSuffix(c.opts.BucketPath, "/"), "/")
	if p == "" {
		return name
	}
	return p + "/" + name
}

// makeFileURL constructs the public URL for the file based on the provided settings.
func (c *Client) makeFileURL(name string) string {
	if c.opts.PublicURL != "" {
		return c.opts.PublicURL + "/" + c.makeBucketPath(name)
	}

	return c.opts.URL + "/" + c.opts.Bucket + "/" + c.makeBucketPath(name)
}

// Name returns the name of the storage implementation, which is "s3" in this case.
func (c *Client) Name() string {
	return "s3"
}

// SignedURLValidator returns nil as S3 handles its own presigned URL validation.
// The S3 service validates presigned URLs when they are accessed.
func (c *Client) SignedURLValidator() func(name, sig string, exp int64) bool {
	return nil
}
