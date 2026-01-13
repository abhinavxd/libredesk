package email

import (
	"fmt"
	"net/smtp"
)

// xoauth2IMAPClient implements SASL XOAUTH2 authentication for IMAP.
type xoauth2IMAPClient struct {
	username string
	token    string
}

// Start begins the XOAUTH2 authentication.
// Returns raw bytes - the go-imap library handles base64 encoding.
func (c *xoauth2IMAPClient) Start() (string, []byte, error) {
	authString := fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", c.username, c.token)
	return "XOAUTH2", []byte(authString), nil
}

// Next continues the authentication.
// For XOAUTH2, we return an empty response to acknowledge the server's challenge.
func (c *xoauth2IMAPClient) Next(challenge []byte) ([]byte, error) {
	return nil, nil
}

// XOAuth2SMTPAuth implements smtp.Auth for XOAUTH2 authentication.
type XOAuth2SMTPAuth struct {
	Username string
	Token    string
}

// Start begins the XOAUTH2 authentication for SMTP.
// Returns raw bytes - the net/smtp library handles base64 encoding.
func (a *XOAuth2SMTPAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	authString := fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", a.Username, a.Token)
	return "XOAUTH2", []byte(authString), nil
}

// Next continues the SMTP authentication.
func (a *XOAuth2SMTPAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		// XOAUTH2 typically completes in one round
		return nil, fmt.Errorf("unexpected server challenge in XOAUTH2")
	}
	return nil, nil
}
