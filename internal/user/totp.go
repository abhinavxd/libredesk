package user

import (
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/pquerna/otp/totp"
	"github.com/volatiletech/null/v9"
)

type totpRow struct {
	Secret  null.String `db:"totp_secret"`
	Enabled bool        `db:"totp_enabled"`
}

func (u *Manager) TOTPEnabled(userID int) (bool, error) {
	var row totpRow
	if err := u.q.GetTOTPSecret.Get(&row, userID); err != nil {
		return false, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return row.Enabled, nil
}

func (u *Manager) BeginTOTPEnroll(userID int, email string) (string, string, error) {
	enabled, err := u.TOTPEnabled(userID)
	if err != nil {
		return "", "", err
	}
	if enabled {
		return "", "", envelope.NewError(envelope.InputError, u.i18n.T("auth.totp.alreadyEnabled"), nil)
	}
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Stackblaze",
		AccountName: email,
	})
	if err != nil {
		u.lo.Error("error generating totp secret", "error", err)
		return "", "", envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if _, err := u.q.SetTOTPSecret.Exec(userID, key.Secret(), false); err != nil {
		u.lo.Error("error storing totp secret", "error", err)
		return "", "", envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return key.Secret(), key.URL(), nil
}

func (u *Manager) ConfirmTOTP(userID int, code string) error {
	var row totpRow
	if err := u.q.GetTOTPSecret.Get(&row, userID); err != nil || !row.Secret.Valid {
		return envelope.NewError(envelope.InputError, u.i18n.T("auth.totp.notStarted"), nil)
	}
	if !totp.Validate(code, row.Secret.String) {
		return envelope.NewError(envelope.InputError, u.i18n.T("auth.totp.invalid"), nil)
	}
	if _, err := u.q.SetTOTPSecret.Exec(userID, row.Secret.String, true); err != nil {
		return envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

func (u *Manager) VerifyTOTP(userID int, code string) error {
	var row totpRow
	if err := u.q.GetTOTPSecret.Get(&row, userID); err != nil || !row.Enabled || !row.Secret.Valid {
		return envelope.NewError(envelope.InputError, u.i18n.T("auth.totp.invalid"), nil)
	}
	if !totp.Validate(code, row.Secret.String) {
		return envelope.NewError(envelope.InputError, u.i18n.T("auth.totp.invalid"), nil)
	}
	return nil
}

func (u *Manager) DisableTOTP(userID int) error {
	if _, err := u.q.SetTOTPSecret.Exec(userID, nil, false); err != nil {
		return envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}
