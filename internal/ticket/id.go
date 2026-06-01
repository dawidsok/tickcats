// id.go handles stable ticket identifier generation and validation.
// IDs use the format "TC-XXXXXX" where X is drawn from a Crockford-inspired
// alphabet that omits I, L, O, and U to avoid visual ambiguity.
// Uniqueness is enforced by checking against the caller-supplied set of
// existing IDs; up to GenerateIDMaxAttempts retries are made before giving up.
package ticket

import (
	"crypto/rand"
	"fmt"
	"strings"
)

const (
	IDPrefix              = "TC-"
	IDSuffixLen           = 6
	IDAlphabet            = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	GenerateIDMaxAttempts = 100
)

// ValidID reports whether id is a well-formed TC-XXXXXX identifier.
func ValidID(id string) bool {
	if len(id) != len(IDPrefix)+IDSuffixLen || !strings.HasPrefix(id, IDPrefix) {
		return false
	}
	for _, r := range id[len(IDPrefix):] {
		if !strings.ContainsRune(IDAlphabet, r) {
			return false
		}
	}
	return true
}

// GenerateID generates a random TC-XXXXXX identifier not already present in
// existing. Returns an error if a unique ID cannot be found within
// GenerateIDMaxAttempts attempts (collision probability is negligible at scale).
func GenerateID(existing map[string]bool) (string, error) {
	for range GenerateIDMaxAttempts {
		id, err := randomID()
		if err != nil {
			return "", err
		}
		if !existing[id] {
			return id, nil
		}
	}
	return "", fmt.Errorf("could not generate unique ticket id after %d attempts", GenerateIDMaxAttempts)
}

func randomID() (string, error) {
	buf := make([]byte, IDSuffixLen)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate ticket id: %w", err)
	}
	var b strings.Builder
	b.WriteString(IDPrefix)
	for _, v := range buf {
		b.WriteByte(IDAlphabet[int(v)%len(IDAlphabet)])
	}
	return b.String(), nil
}
