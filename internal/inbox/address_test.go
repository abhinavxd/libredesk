package inbox

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/stretchr/testify/require"
)

func TestValidateEmailAddresses(t *testing.T) {
	primary, aliases, err := ValidateEmailAddresses("Support <SUPPORT@example.com>", models.EmailAliases{{Email: "Billing@Example.com"}, {Email: "sales@example.com"}})
	require.NoError(t, err)
	require.Equal(t, "support@example.com", primary)
	require.Equal(t, models.EmailAliases{{Email: "billing@example.com", VerificationStatus: models.AliasVerificationNotVerified}, {Email: "sales@example.com", VerificationStatus: models.AliasVerificationNotVerified}}, aliases)
	_, _, err = ValidateEmailAddresses("support@company.com", models.EmailAliases{{Email: "support@otherbrand.com"}})
	require.NoError(t, err)

	for name, values := range map[string][]string{
		"primary repeated": {"support@example.com"},
		"case duplicate":   {"billing@example.com", "BILLING@example.com"},
		"display name":     {"Billing <billing@example.com>"},
	} {
		t.Run(name, func(t *testing.T) {
			input := make(models.EmailAliases, len(values))
			for i, value := range values {
				input[i].Email = value
			}
			_, _, err := ValidateEmailAddresses("support@example.com", input)
			require.Error(t, err)
		})
	}
}

func TestSendableEmailAddressesRequiresVerification(t *testing.T) {
	addresses, err := SendableEmailAddresses("support@company.com", models.EmailAliases{
		{Email: "receive@otherbrand.com"},
		{Email: "billing@company.com", VerificationStatus: models.AliasVerificationVerified},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"support@company.com", "billing@company.com"}, addresses)
}
