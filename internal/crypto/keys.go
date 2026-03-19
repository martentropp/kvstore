package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"os"
)

type KeyPair struct {
    Public  ed25519.PublicKey
    Private ed25519.PrivateKey
}

func GenerateKeyPair() (*KeyPair, error) {
    pub, priv, err := ed25519.GenerateKey(rand.Reader)

	// verify no errors happened
    if err != nil {
        return nil, err
    }
	
    return &KeyPair{Public: pub, Private: priv}, nil
}

func (kp *KeyPair) SaveToFile(path string) error {
	// permissions set so only the owner can read it
    return os.WriteFile(path, []byte(hex.EncodeToString(kp.Private)), 0600)
}

func LoadFromFile(path string) (*KeyPair, error) {
    data, err := os.ReadFile(path)

	// verify reading success
    if err != nil {
        return nil, err
    }

	// verify decoding success
    privBytes, err := hex.DecodeString(string(data))
    if err != nil {
        return nil, err
    }

	// get key values
    priv := ed25519.PrivateKey(privBytes)
    pub := priv.Public().(ed25519.PublicKey)

    return &KeyPair{Public: pub, Private: priv}, nil
}