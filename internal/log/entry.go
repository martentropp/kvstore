package log

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"kvstore/internal/crypto"
)

type LogEntry struct {
    Index     uint64 `json:"index"`
    Key       string `json:"key"`
    Value     string `json:"value"`
    NodeID    string `json:"node_id"`
    Timestamp int64  `json:"timestamp"`
    PrevHash  string `json:"prev_hash"`
    Signature string `json:"signature"`
}

// hash returns the SHA-256 hash of this entry's contents, excluding the signature
func (e *LogEntry) Hash() string {
    payload := fmt.Sprintf("%d|%s|%s|%s|%d|%s",
        e.Index, e.Key, e.Value, e.NodeID, e.Timestamp, e.PrevHash)
    h := sha256.Sum256([]byte(payload))

    return hex.EncodeToString(h[:])
}

func (e *LogEntry) Sign(kp *crypto.KeyPair) {
    payload := fmt.Sprintf("%d|%s|%s|%s|%d|%s",
        e.Index, e.Key, e.Value, e.NodeID, e.Timestamp, e.PrevHash)
    sig := crypto.Sign(kp.Private, []byte(payload))
    e.Signature = hex.EncodeToString(sig)
}

func (e *LogEntry) Verify(pub ed25519.PublicKey) bool {
    payload := fmt.Sprintf("%d|%s|%s|%s|%d|%s",
        e.Index, e.Key, e.Value, e.NodeID, e.Timestamp, e.PrevHash)
    sig, err := hex.DecodeString(e.Signature)
	
    if err != nil {
        return false
    }

    return crypto.Verify(pub, []byte(payload), sig)
}

func (e *LogEntry) ToJSON() ([]byte, error) {
    return json.Marshal(e)
}