package media

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (m *Manager) mediaSignature(name string, exp int64) string {
	mac := hmac.New(sha256.New, []byte(m.signingKey))
	fmt.Fprintf(mac, "%s:%d", name, exp)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (m *Manager) validateMediaSignature(name, sig string, exp int64) bool {
	return time.Now().Unix() <= exp && hmac.Equal([]byte(sig), []byte(m.mediaSignature(name, exp)))
}

func (m *Manager) signedMediaURL(name string) string {
	if _, err := uuid.Parse(strings.TrimPrefix(name, "thumb_")); err != nil {
		return ""
	}
	expiry := m.urlExpiry
	if expiry <= 0 {
		expiry = time.Hour
	}
	exp := time.Now().Add(expiry).Unix()
	return fmt.Sprintf("%s%s/%s?sig=%s&exp=%d", strings.TrimRight(m.rootURL(), "/"), PublicURI, name, m.mediaSignature(strings.TrimPrefix(name, "thumb_"), exp), exp)
}

func (m *Manager) Open(name string) (io.ReadCloser, error) {
	return m.store.Open(name)
}
