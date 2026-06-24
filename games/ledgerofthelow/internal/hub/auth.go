package hub

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// newAPIKey returns a fresh high-entropy API key and its storage hash. The hub
// stores only the hash; the plaintext key is shown to the node once at registration.
func newAPIKey() (key, hash string) {
	var b [32]byte
	_, _ = rand.Read(b[:])
	key = base64.RawURLEncoding.EncodeToString(b[:])
	return key, hashKey(key)
}

// hashKey hashes an API key for storage/lookup. API keys are high-entropy, so a
// fast SHA-256 is appropriate (unlike user passwords, which use bcrypt).
func hashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}
