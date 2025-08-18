package fs

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/abhinavxd/libredesk/internal/media"
)

// Opts holds fs options.
type Opts struct {
	UploadPath string
	UploadURI  string
	RootURL    string
}

// Client implements `media.Store`
type Client struct {
	opts Opts
}

// New initialises store for Filesystem provider.
func New(opts Opts) (media.Store, error) {
	return &Client{
		opts: opts,
	}, nil
}

// Put accepts the filename, the content type and file object itself and stores the file in disk.
func (c *Client) Put(filename string, cType string, src io.ReadSeeker) (string, error) {
	var out *os.File

	// Get the directory path
	dir := getDir(c.opts.UploadPath)
	o, err := os.OpenFile(filepath.Join(dir, filename), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
	if err != nil {
		return "", err
	}
	out = o
	defer out.Close()

	if _, err := io.Copy(out, src); err != nil {
		return "", err
	}
	return filename, nil
}

// GetURL accepts a filename and retrieves the full URL for file.
func (c *Client) GetURL(name string) string {
	return fmt.Sprintf("%s%s/%s", c.opts.RootURL, c.opts.UploadURI, name)
}

// GetBlob accepts a URL, reads the file, and returns the blob.
func (c *Client) GetBlob(url string) ([]byte, error) {
	b, err := os.ReadFile(filepath.Join(getDir(c.opts.UploadPath), filepath.Base(url)))
	return b, err
}

// Delete accepts a filename and removes it from disk.
func (c *Client) Delete(file string) error {
	dir := getDir(c.opts.UploadPath)
	err := os.Remove(filepath.Join(dir, file))
	if err != nil {
		return err
	}
	return nil
}

// Name returns the name of the store.
func (c *Client) Name() string {
	return "fs"
}

// GetSignedURL generates a signed URL for the file with expiration.
// This implements the SignedURLStore interface for secure public access.
func (c *Client) GetSignedURL(name string, expiresAt time.Time, secret []byte) string {
	// Generate base URL
	baseURL := c.GetURL(name)
	
	// Create the signature payload: name + expires timestamp
	expires := expiresAt.Unix()
	payload := name + strconv.FormatInt(expires, 10)
	
	// Generate HMAC-SHA256 signature
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(payload))
	signature := base64.URLEncoding.EncodeToString(h.Sum(nil))
	
	// Parse base URL and add query parameters
	u, err := url.Parse(baseURL)
	if err != nil {
		// Fallback to base URL if parsing fails
		return baseURL
	}
	
	// Add signature and expires parameters
	query := u.Query()
	query.Set("signature", signature)
	query.Set("expires", strconv.FormatInt(expires, 10))
	u.RawQuery = query.Encode()
	
	return u.String()
}

// VerifySignature verifies that a signature is valid for the given parameters.
// This implements the SignedURLStore interface for secure public access.
func (c *Client) VerifySignature(name, signature string, expiresAt time.Time, secret []byte) bool {
	// Check if URL has expired
	if time.Now().After(expiresAt) {
		return false
	}
	
	// Recreate the signature payload: name + expires timestamp
	expires := expiresAt.Unix()
	payload := name + strconv.FormatInt(expires, 10)
	
	// Generate expected HMAC-SHA256 signature
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(payload))
	expectedSignature := base64.URLEncoding.EncodeToString(h.Sum(nil))
	
	// Use constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSignature)) == 1
}

// getDir returns the current working directory path if no directory is specified,
// else returns the directory path specified itself.
func getDir(dir string) string {
	if dir == "" {
		dir, _ = os.Getwd()
	}
	return dir
}
