package handlers

import (
	"fmt"
	"github.com/Crowley723/conduit/internal/middlewares"
	"strings"
)

// RedactEmail is used to redact emails (mostly for logs)
func RedactEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}

	localRunes := []rune(parts[0])
	domain := parts[1]

	if len(localRunes) <= 2 {
		return strings.Repeat("*", len(localRunes)) + "@" + domain
	}

	first := string(localRunes[0])
	last := string(localRunes[len(localRunes)-1])
	middle := strings.Repeat("*", len(localRunes)-2)

	return first + middle + last + "@" + domain
}

// deriveCommonName gets a CN for the user's mTLS certificate
func deriveCommonName(principal middlewares.Principal) string {
	if principal.GetDisplayName() != "" {
		return principal.GetDisplayName()
	}

	if principal.GetUsername() != "" {
		return principal.GetUsername()
	}

	if principal.GetEmail() != "" {
		return principal.GetEmail()
	}

	issuerDomain := strings.TrimPrefix(principal.GetIss(), "https://")
	issuerDomain = strings.TrimPrefix(issuerDomain, "http://")
	return fmt.Sprintf("%s@%s", principal.GetSub(), issuerDomain)
}
