package log

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"kvstore/internal/crypto"
	"sync"
	"time"
)

type Chain struct {
    mu      sync.RWMutex
    entries []*LogEntry
    nodeID  string
    keyPair *crypto.KeyPair
}

func NewChain(nodeID string, kp *crypto.KeyPair) *Chain {
    return &Chain{
        entries: make([]*LogEntry, 0),
        nodeID:  nodeID,
        keyPair: kp,
    }
}

func (c *Chain) Append(key, value string) *LogEntry {
	// prevent concurrent writes from separate nodes
    c.mu.Lock()
    defer c.mu.Unlock()

    prevHash := "genesis"
    if len(c.entries) > 0 {
        prevHash = c.entries[len(c.entries) - 1].Hash()
    }

    entry := &LogEntry{
        Index:     uint64(len(c.entries)),
        Key:       key,
        Value:     value,
        NodeID:    c.nodeID,
        Timestamp: time.Now().UnixNano(),
        PrevHash:  prevHash,
    }
    entry.Sign(c.keyPair)

    c.entries = append(c.entries, entry)
    return entry
}

func (c *Chain) Verify(trustedKeys map[string]ed25519.PublicKey) error {
    c.mu.RLock()
    defer c.mu.RUnlock()

    for i, entry := range c.entries {
        // check signature
        pub, ok := trustedKeys[entry.NodeID]
        if !ok {
            return errors.New("unknown node: " + entry.NodeID)
        }

        if !entry.Verify(pub) {
            return errors.New("invalid signature at entry " + entry.NodeID)
        }

        // check chain linkage
        expectedPrev := "genesis"
        if i > 0 {
            expectedPrev = c.entries[i - 1].Hash()
        }

        if entry.PrevHash != expectedPrev {
            return fmt.Errorf("broken chain at index %d", i)
        }
    }

    return nil
}

func (c *Chain) Entries() []*LogEntry {
    c.mu.RLock()
    defer c.mu.RUnlock()

    return c.entries
}

func (c *Chain) AcceptEntry(entry *LogEntry, trustedKeys map[string]ed25519.PublicKey) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    // verify signature
    pub, ok := trustedKeys[entry.NodeID]
    if !ok {
        return errors.New("unknown node: " + entry.NodeID)
    }
    if !entry.Verify(pub) {
        return errors.New("invalid signature")
    }

    // verify chain linkage
    expectedPrev := "genesis"
    if len(c.entries) > 0 {
        expectedPrev = c.entries[len(c.entries) - 1].Hash()
    }
	
    if entry.PrevHash != expectedPrev {
        return errors.New("chain linkage broken")
    }

    c.entries = append(c.entries, entry)
    return nil
}