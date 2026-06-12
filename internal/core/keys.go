package core

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

// RealityKeypair holds a base64-url X25519 keypair plus a short id, matching the
// output of `xray x25519`. Used to seed VLESS+Reality inbounds.
type RealityKeypair struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
	ShortID    string `json:"short_id"`
}

// GenerateRealityKeypair produces a fresh X25519 keypair (clamped private key)
// and a random 8-byte short id.
func GenerateRealityKeypair() (RealityKeypair, error) {
	var priv [32]byte
	if _, err := rand.Read(priv[:]); err != nil {
		return RealityKeypair{}, err
	}
	// Clamp per X25519.
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64

	pub, err := curve25519.X25519(priv[:], curve25519.Basepoint)
	if err != nil {
		return RealityKeypair{}, err
	}

	shortID, err := randomHex(8)
	if err != nil {
		return RealityKeypair{}, err
	}

	return RealityKeypair{
		PrivateKey: base64.RawURLEncoding.EncodeToString(priv[:]),
		PublicKey:  base64.RawURLEncoding.EncodeToString(pub),
		ShortID:    shortID,
	}, nil
}

// NewUUID returns a random RFC-4122 v4 UUID string (VLESS/VMess/TUIC client id).
func NewUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// RandomSecret returns a URL-safe random secret (Trojan/Shadowsocks/Hysteria2
// password, subscription token).
func RandomSecret(nbytes int) (string, error) {
	b := make([]byte, nbytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
