package models

import (
	"net/mail"
	"regexp"
)

// e164Pattern matches E.164 phone numbers: a leading '+', 1-15 digits total,
// no leading zero after the '+'.
var e164Pattern = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)

// isValidEmail reports whether email is a single, addr-spec-only RFC 5322
// address (no display name, no comment).
func isValidEmail(email string) bool {
	if email == "" {
		return false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	return addr.Address == email
}

// isValidE164 reports whether phone is a valid E.164 number.
func isValidE164(phone string) bool {
	return e164Pattern.MatchString(phone)
}
