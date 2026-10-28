package user

import (
	"bytes"
	"strings"
	"testing"

	"github.com/zerodha/logf"
	"golang.org/x/crypto/bcrypt"
)

func TestPasswordMismatchDoesNotLogApplicationError(t *testing.T) {
	var logs bytes.Buffer
	lo := logf.New(logf.Opts{Writer: &logs})
	manager := &Manager{lo: &lo}
	hash, err := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.verifyPassword([]byte("wrong"), string(hash)); err == nil {
		t.Fatal("incorrect password accepted")
	}
	if logs.Len() != 0 {
		t.Fatalf("ordinary password mismatch logged: %s", logs.String())
	}
	if err := manager.verifyPassword([]byte("correct"), "invalid stored hash"); err == nil {
		t.Fatal("corrupt password hash accepted")
	}
	if !strings.Contains(logs.String(), "level=error") {
		t.Fatal("corrupt stored hash has no application error log")
	}
}
