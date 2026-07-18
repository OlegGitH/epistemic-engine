package epistemic

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// CanonicalJSON produces the v0.1 canonical representation: UTF-8 JSON with
// lexicographically sorted object keys, no insignificant whitespace, and JSON
// number spellings preserved through json.Number. Protocol proof fixtures avoid
// non-integer numbers so this representation is reproducible across SDKs.
func CanonicalJSON(value any) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var normalized any
	if err = decoder.Decode(&normalized); err != nil {
		return nil, err
	}
	canonical, err := json.Marshal(normalized)
	if err != nil {
		return nil, err
	}
	return canonical, nil
}

func Hash(value any) (string, error) {
	canonical, err := CanonicalJSON(value)
	if err != nil {
		return "", fmt.Errorf("canonicalize: %w", err)
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}

func VerifyHash(value any, expected string) error {
	actual, err := Hash(value)
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("hash mismatch: got %s want %s", actual, expected)
	}
	return nil
}
