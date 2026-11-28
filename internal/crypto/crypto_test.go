package crypto

import (
	"strings"
	"testing"
)

const testKey = "12345678901234567890123456789012" // 32 bytes

func TestEncryptDecryptRoundtrip(t *testing.T) {
	plaintext := "hello, libredesk!"
	encrypted, err := Encrypt(plaintext, testKey)
	if err != nil {
		t.Fatalf("Encrypt returned error: %v", err)
	}
	if !IsEncrypted(encrypted) {
		t.Fatalf("expected encrypted value to have prefix %q, got %q", EncryptedPrefix, encrypted)
	}
	decrypted, err := Decrypt(encrypted, testKey)
	if err != nil {
		t.Fatalf("Decrypt returned error: %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("roundtrip mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptProducesUniqueOutputs(t *testing.T) {
	// AES-GCM uses a random nonce, so the same plaintext should encrypt differently each time.
	plaintext := "same input"
	a, err := Encrypt(plaintext, testKey)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Encrypt(plaintext, testKey)
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Error("expected different ciphertexts for the same plaintext due to random nonce")
	}
}

func TestEncryptEmptyPlaintext(t *testing.T) {
	result, err := Encrypt("", testKey)
	if err != nil {
		t.Fatalf("Encrypt(\"\") returned error: %v", err)
	}
	if result != "" {
		t.Errorf("Encrypt(\"\") = %q, want empty string", result)
	}
}

func TestEncryptAlreadyEncrypted(t *testing.T) {
	plaintext := "secret"
	encrypted, _ := Encrypt(plaintext, testKey)
	// Encrypting an already-encrypted value should return it unchanged.
	again, err := Encrypt(encrypted, testKey)
	if err != nil {
		t.Fatalf("re-encrypt returned error: %v", err)
	}
	if again != encrypted {
		t.Errorf("re-encrypting changed the value: got %q, want %q", again, encrypted)
	}
}

func TestEncryptInvalidKeyLength(t *testing.T) {
	_, err := Encrypt("data", "tooshort")
	if err != ErrInvalidKey {
		t.Errorf("expected ErrInvalidKey, got %v", err)
	}
}

func TestDecryptEmptyCiphertext(t *testing.T) {
	result, err := Decrypt("", testKey)
	if err != nil {
		t.Fatalf("Decrypt(\"\") returned error: %v", err)
	}
	if result != "" {
		t.Errorf("Decrypt(\"\") = %q, want empty string", result)
	}
}

func TestDecryptUnencryptedPassthrough(t *testing.T) {
	// Values without the prefix are returned as-is.
	plain := "not-encrypted"
	result, err := Decrypt(plain, testKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != plain {
		t.Errorf("got %q, want %q", result, plain)
	}
}

func TestDecryptInvalidKeyLength(t *testing.T) {
	_, err := Decrypt(EncryptedPrefix+"somedata", "short")
	if err != ErrInvalidKey {
		t.Errorf("expected ErrInvalidKey, got %v", err)
	}
}

func TestDecryptInvalidBase64(t *testing.T) {
	_, err := Decrypt(EncryptedPrefix+"!!!not-base64!!!", testKey)
	if err != ErrInvalidCiphertext {
		t.Errorf("expected ErrInvalidCiphertext, got %v", err)
	}
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	encrypted, _ := Encrypt("secret", testKey)
	// Flip the last character to corrupt the ciphertext.
	tampered := encrypted[:len(encrypted)-1] + "X"
	if strings.HasSuffix(encrypted, "X") {
		tampered = encrypted[:len(encrypted)-1] + "Y"
	}
	_, err := Decrypt(tampered, testKey)
	if err == nil {
		t.Error("expected error decrypting tampered ciphertext, got nil")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	encrypted, _ := Encrypt("secret", testKey)
	otherKey := "99999999999999999999999999999999" // 32 bytes, different
	_, err := Decrypt(encrypted, otherKey)
	if err != ErrDecryptionFailed {
		t.Errorf("expected ErrDecryptionFailed, got %v", err)
	}
}

func TestIsEncrypted(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{EncryptedPrefix + "somedata", true},
		{"plaintext", false},
		{"", false},
		{"enc", false},
		{EncryptedPrefix, true},
	}
	for _, tc := range tests {
		if got := IsEncrypted(tc.input); got != tc.want {
			t.Errorf("IsEncrypted(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}
