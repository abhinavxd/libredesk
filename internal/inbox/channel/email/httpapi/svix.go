package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// svixTolerance is the maximum age of a webhook timestamp we'll accept, guarding against replay.
const svixTolerance = 5 * time.Minute

// verifySvixSignature implements the Svix webhook signing scheme (used by Resend and several
// other providers): HMAC-SHA256 over "{id}.{timestamp}.{body}", keyed by the base64-decoded
// secret (after stripping the "whsec_" prefix), base64-encoded, compared against one or more
// "v1,<sig>" values in the svix-signature header.
func verifySvixSignature(secret string, headers http.Header, body []byte) error {
	msgID := headers.Get("svix-id")
	msgTimestamp := headers.Get("svix-timestamp")
	sigHeader := headers.Get("svix-signature")
	if msgID == "" || msgTimestamp == "" || sigHeader == "" {
		return fmt.Errorf("missing svix signature headers")
	}

	ts, err := strconv.ParseInt(msgTimestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid svix-timestamp: %w", err)
	}
	if age := time.Since(time.Unix(ts, 0)); age > svixTolerance || age < -svixTolerance {
		return fmt.Errorf("svix-timestamp outside tolerance")
	}

	secretBytes, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(secret, "whsec_"))
	if err != nil {
		return fmt.Errorf("invalid webhook secret encoding: %w", err)
	}

	signedContent := fmt.Sprintf("%s.%s.%s", msgID, msgTimestamp, body)
	mac := hmac.New(sha256.New, secretBytes)
	mac.Write([]byte(signedContent))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	for _, candidate := range strings.Fields(sigHeader) {
		parts := strings.SplitN(candidate, ",", 2)
		if len(parts) != 2 {
			continue
		}
		if hmac.Equal([]byte(parts[1]), []byte(expected)) {
			return nil
		}
	}
	return fmt.Errorf("signature mismatch")
}
