package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// keys are derived from SESSION_KEY so one secret covers both jobs without
// reusing the same key for signing and encryption.
type keys struct {
	sign []byte
	aead cipher.AEAD
}

func deriveKeys(master []byte) (keys, error) {
	if len(master) < 32 {
		return keys{}, fmt.Errorf("auth: session key must be at least 32 bytes, got %d", len(master))
	}
	sign, err := hkdf.Key(sha256.New, master, nil, "resonance session signing", 32)
	if err != nil {
		return keys{}, err
	}
	enc, err := hkdf.Key(sha256.New, master, nil, "resonance token encryption", 32)
	if err != nil {
		return keys{}, err
	}
	block, err := aes.NewCipher(enc)
	if err != nil {
		return keys{}, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return keys{}, err
	}
	return keys{sign: sign, aead: aead}, nil
}

// seal encrypts plaintext with AES-GCM, binding it to userID so a token
// can't be moved to another user's row.
func (k keys) seal(userID, plaintext string) []byte {
	nonce := make([]byte, k.aead.NonceSize())
	rand.Read(nonce)
	return k.aead.Seal(nonce, nonce, []byte(plaintext), []byte(userID))
}

func (k keys) open(userID string, sealed []byte) (string, error) {
	n := k.aead.NonceSize()
	if len(sealed) < n {
		return "", errors.New("auth: sealed token too short")
	}
	pt, err := k.aead.Open(nil, sealed[:n], sealed[n:], []byte(userID))
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

// signSession returns a cookie value of the form
// base64url(userID).expiryUnix.base64url(hmac).
func (k keys) signSession(userID string, expiry time.Time) string {
	payload := base64.RawURLEncoding.EncodeToString([]byte(userID)) + "." + strconv.FormatInt(expiry.Unix(), 10)
	return payload + "." + base64.RawURLEncoding.EncodeToString(k.mac(payload))
}

// verifySession returns the user ID from a cookie value if the signature is
// valid and it hasn't expired.
func (k keys) verifySession(value string, now time.Time) (string, bool) {
	i := strings.LastIndexByte(value, '.')
	if i < 0 {
		return "", false
	}
	payload, sig := value[:i], value[i+1:]
	got, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil || !hmac.Equal(got, k.mac(payload)) {
		return "", false
	}

	rawID, rawExp, ok := strings.Cut(payload, ".")
	if !ok {
		return "", false
	}
	exp, err := strconv.ParseInt(rawExp, 10, 64)
	if err != nil || !now.Before(time.Unix(exp, 0)) {
		return "", false
	}
	id, err := base64.RawURLEncoding.DecodeString(rawID)
	if err != nil || len(id) == 0 {
		return "", false
	}
	return string(id), true
}

func (k keys) mac(payload string) []byte {
	m := hmac.New(sha256.New, k.sign)
	m.Write([]byte(payload))
	return m.Sum(nil)
}
