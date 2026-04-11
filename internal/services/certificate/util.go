package certificate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// GenerateCertificateName generates a unique certificate name based on owner and timestamp
// Format: hash of "sub:iss:timestamp" truncated to fit Kubernetes naming constraints
func GenerateCertificateName(sub, iss string, timestamp time.Time) string {
	input := fmt.Sprintf("%s:%s:%d", sub, iss, timestamp.Unix())
	hash := sha256.Sum256([]byte(input))
	hashStr := hex.EncodeToString(hash[:])

	// Kubernetes DNS-1123 subdomain naming requirements:
	// - lowercase alphanumeric characters, '-' or '.'
	// - must start and end with alphanumeric
	// - max 253 characters
	// Use "cert-" prefix + first 32 chars of hash
	name := fmt.Sprintf("cert-%s", hashStr[:32])
	return strings.ToLower(name)
}
