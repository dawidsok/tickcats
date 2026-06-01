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
