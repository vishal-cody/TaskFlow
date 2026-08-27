package validator

import (
	"testing"
)

func TestValidateEmail(t *testing.T) {
	valid := []string{
		"user@example.com",
		"test.user@domain.co",
		"user+tag@example.org",
	}
	for _, email := range valid {
		if err := ValidateEmail(email); err != nil {
			t.Errorf("expected %q to be valid email, got: %v", email, err)
		}
	}

	invalid := []string{
		"",
		"notanemail",
		"@missing-local.com",
		"missing-domain@",
	}
	for _, email := range invalid {
		if err := ValidateEmail(email); err == nil {
			t.Errorf("expected %q to be invalid email", email)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword("secure-password-123"); err != nil {
		t.Errorf("expected long password to be valid, got: %v", err)
	}
	if err := ValidatePassword("12345678"); err != nil {
		t.Errorf("expected 8-char password to be valid, got: %v", err)
	}

	if err := ValidatePassword(""); err == nil {
		t.Error("expected empty password to be invalid")
	}
	if err := ValidatePassword("short"); err == nil {
		t.Error("expected short password to be invalid")
	}
	if err := ValidatePassword("1234567"); err == nil {
		t.Error("expected 7-char password to be invalid")
	}
}
