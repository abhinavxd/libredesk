// Package image provides utilities for processing image files, including
// retrieving image dimensions and creating thumbnails.
package image

import (
	"bytes"
	"io"

	"github.com/disintegration/imaging"
	"github.com/gabriel-vasile/mimetype"
)

var (
	Exts         = []string{"gif", "png", "jpg", "jpeg"}
	DefThumbSize = 150
	ThumbPrefix  = "thumb_"
)

// IsImageByContent returns true when the file's magic bytes identify it as one
// of the raster formats this package can decode. Used as a fallback when the
// filename has no extension or an unreliable one (e.g. attachments arriving
// through email without proper file extensions).
func IsImageByContent(r io.ReadSeeker) bool {
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return false
	}
	defer r.Seek(0, io.SeekStart)
	mtype, err := mimetype.DetectReader(r)
	if err != nil {
		return false
	}
	switch mtype.String() {
	case "image/png", "image/jpeg", "image/gif":
		return true
	}
	return false
}

// GetDimensions returns the width and height of the image in the provided file.
// It returns an error if the image cannot be decoded.
func GetDimensions(r io.Reader) (int, int, error) {
	img, err := imaging.Decode(r)
	if err != nil {
		return 0, 0, err
	}

	bounds := img.Bounds()
	width := bounds.Max.X
	height := bounds.Max.Y

	return width, height, nil
}

// CreateThumb generates a thumbnail of the given image file with the specified maximum dimension.
// The thumbnail's width will be resized to `thumbPxSize` while maintaining the aspect ratio.
func CreateThumb(thumbPxSize int, r io.Reader) (*bytes.Reader, error) {
	img, err := imaging.Decode(r)
	if err != nil {
		return nil, err
	}

	thumb := imaging.Resize(img, thumbPxSize, 0, imaging.Lanczos)
	var out bytes.Buffer
	if err := imaging.Encode(&out, thumb, imaging.PNG); err != nil {
		return nil, err
	}

	return bytes.NewReader(out.Bytes()), nil
}
