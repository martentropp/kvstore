package log

import (
	"crypto/ed25519"
	"kvstore/internal/crypto"
	"testing"
)

func TestChainTampering(t *testing.T) {
    kp, _ := crypto.GenerateKeyPair()
    chain := NewChain("node1", kp)

    chain.Append("name", "marten")
    chain.Append("city", "boston")
    chain.Append("lang", "go")

    trusted := map[string]ed25519.PublicKey {
        "node1": kp.Public,
    }

    // verify clean
    if err := chain.Verify(trusted); err != nil {
        t.Fatalf("clean chain failed verification: %v", err)
    }

    // tamper with the middle entry
    chain.entries[1].Value = "tallinn"

    if err := chain.Verify(trusted); err == nil {
        t.Fatal("tampered chain should have failed verification")
    }
}