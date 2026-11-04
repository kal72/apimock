package usecase

import (
	"strings"
	"testing"
)

func TestPasswordHashing(t *testing.T) {
	password := "correct horse battery staple"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	second, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	if hash == second || strings.Contains(hash, password) {
		t.Fatal("passwords must use unique salts and never be stored as plaintext")
	}
	if !VerifyPassword(hash, password) || VerifyPassword(hash, "wrong password") {
		t.Fatal("password verification failed")
	}
	for _, bad := range []string{"", "malformed", "pbkdf2-sha256$1$anything$anything"} {
		if VerifyPassword(bad, password) {
			t.Fatal("malformed hash accepted")
		}
	}
	if _, err = HashPassword("short"); err == nil {
		t.Fatal("short password accepted")
	}
	if _, err = HashPassword(strings.Repeat("x", 1025)); err == nil {
		t.Fatal("oversized password accepted")
	}
}

func TestPasswordMinimumEightCharacters(t *testing.T) {
	for _, password := range []string{"1234567", "密碼測試"} {
		if _, err := HashPassword(password); err == nil {
			t.Fatalf("short password accepted: %q", password)
		}
	}
	for _, password := range []string{"12345678", "密碼測試八個字符"} {
		hash, err := HashPassword(password)
		if err != nil || !VerifyPassword(hash, password) {
			t.Fatalf("eight-character password rejected: %v", err)
		}
	}
}
