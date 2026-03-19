package crypto

import (
	"crypto/ed25519"
)

func Sign(priv ed25519.PrivateKey, message []byte) []byte {
    return ed25519.Sign(priv, message)
}

func Verify(pub ed25519.PublicKey, message, sig []byte) bool {
    return ed25519.Verify(pub, message, sig)
}