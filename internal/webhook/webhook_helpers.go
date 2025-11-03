package webhook

import (
	"github.com/abhinavxd/libredesk/internal/crypto"
	"github.com/abhinavxd/libredesk/internal/webhook/models"
)

// encryptSecret encrypts webhook secret if present.
// Returns the encrypted secret or empty string if input is empty.
func (m *Manager) encryptSecret(secret string) (string, error) {
	if secret == "" {
		return "", nil
	}

	encrypted, err := crypto.Encrypt(secret, m.encryptionKey)
	if err != nil {
		m.lo.Error("error encrypting webhook secret", "error", err)
		return "", err
	}

	return encrypted, nil
}

// decryptWebhook decrypts webhook secret in-place.
// Returns nil if decryption is successful or secret is empty.
func (m *Manager) decryptWebhook(webhook *models.Webhook) error {
	if webhook.Secret == "" {
		return nil
	}

	decrypted, err := crypto.Decrypt(webhook.Secret, m.encryptionKey)
	if err != nil {
		m.lo.Error("error decrypting webhook secret", "webhook_id", webhook.ID, "error", err)
		return err
	}

	webhook.Secret = decrypted
	return nil
}

// decryptWebhooks decrypts secrets for a slice of webhooks.
// Continues on error for individual webhooks.
func (m *Manager) decryptWebhooks(webhooks []models.Webhook) {
	for i := range webhooks {
		if err := m.decryptWebhook(&webhooks[i]); err != nil {
			// Continue processing other webhooks even if one fails
			continue
		}
	}
}