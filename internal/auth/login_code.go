package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/redis/go-redis/v9"
)

const (
	loginCodeKeyFmt = "libredesk:oidc_login_code:%s"
	loginCodeTTL    = 5 * time.Minute
	loginCodeLength = 48
)

// loginCode is the handoff between the OIDC callback and the token exchange.
type loginCode struct {
	UserID    int    `json:"user_id"`
	Challenge string `json:"challenge"`
}

// MintLoginCode issues a single-use code bound to a PKCE challenge.
func (a *Auth) MintLoginCode(ctx context.Context, userID int, challenge string) (string, error) {
	code, err := stringutil.RandomAlphanumeric(loginCodeLength)
	if err != nil {
		a.logger.Error("error generating login code", "error", err)
		return "", envelope.NewError(envelope.GeneralError, a.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	payload, err := json.Marshal(loginCode{UserID: userID, Challenge: challenge})
	if err != nil {
		a.logger.Error("error encoding login code", "error", err)
		return "", envelope.NewError(envelope.GeneralError, a.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	if err := a.rd.Set(ctx, fmt.Sprintf(loginCodeKeyFmt, code), payload, loginCodeTTL).Err(); err != nil {
		a.logger.Error("error storing login code", "error", err)
		return "", envelope.NewError(envelope.GeneralError, a.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	return code, nil
}

// ConsumeLoginCode redeems a code exactly once and returns the user it was minted for.
func (a *Auth) ConsumeLoginCode(ctx context.Context, code, verifier string) (int, error) {
	if code == "" || verifier == "" {
		return 0, envelope.NewError(envelope.UnauthorizedError, a.i18n.T("validation.invalidCredential"), nil)
	}

	// GetDel so a replayed code cannot be redeemed a second time.
	payload, err := a.rd.GetDel(ctx, fmt.Sprintf(loginCodeKeyFmt, code)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, envelope.NewError(envelope.UnauthorizedError, a.i18n.T("validation.invalidCredential"), nil)
		}
		a.logger.Error("error reading login code", "error", err)
		return 0, envelope.NewError(envelope.GeneralError, a.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	var stored loginCode
	if err := json.Unmarshal(payload, &stored); err != nil {
		a.logger.Error("error decoding login code", "error", err)
		return 0, envelope.NewError(envelope.GeneralError, a.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	if subtle.ConstantTimeCompare([]byte(stored.Challenge), []byte(PKCEChallenge(verifier))) != 1 {
		return 0, envelope.NewError(envelope.UnauthorizedError, a.i18n.T("validation.invalidCredential"), nil)
	}

	return stored.UserID, nil
}

// PKCEChallenge returns the RFC 7636 S256 challenge for a verifier.
func PKCEChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
