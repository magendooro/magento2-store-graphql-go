package graph

import (
	"net/mail"
	"strings"
)

// isValidEmail returns true if the address is RFC 5322 valid.
func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

// derefStr safely dereferences a *string, returning "" for nil.
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// trimmedNotEmpty returns true when s (after trim) is non-empty.
func trimmedNotEmpty(s string) bool {
	return strings.TrimSpace(s) != ""
}
