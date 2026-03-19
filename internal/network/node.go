package network

import (
	"bufio"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"kvstore/internal/crypto"
	"kvstore/internal/log"
	"kvstore/internal/store"
	"net"
	"sync"
)

type Node struct {
    ID          string
    addr        string
    keyPair     *crypto.KeyPair
    chain       *log.Chain
    store       *store.KVStore
    peers       map[string]string            // nodeID -> address
    trustedKeys map[string]ed25519.PublicKey // nodeID -> pubkey
	logFile     string
    mu          sync.RWMutex
}

func NewNode(id, addr string, kp *crypto.KeyPair) *Node {
    return &Node{
        ID:          id,
        addr:        addr,
        keyPair:     kp,
        chain:       log.NewChain(id, kp),
        store:       store.New(),
        peers:       make(map[string]string),
        trustedKeys: make(map[string]ed25519.PublicKey),
    }
}

func (n *Node) AddPeer(id, addr string, pubKey ed25519.PublicKey) {
    n.mu.Lock()
    defer n.mu.Unlock()

    n.peers[id] = addr
    n.trustedKeys[id] = pubKey

    // trust ourselves
    n.trustedKeys[n.ID] = n.keyPair.Public
}

func (n *Node) Start() error {
    ln, err := net.Listen("tcp", n.addr)

	// failed to listen
    if err != nil {
        return err
    }

	// accept incoming connections
    fmt.Printf("[%s] listening on %s\n", n.ID, n.addr)
    go func() {
        for {
            conn, err := ln.Accept()
            if err != nil {
                continue
            }
            go n.handleConn(conn)
        }
    }()

    return nil
}

func (n *Node) handleConn(conn net.Conn) {
    defer conn.Close()

    scanner := bufio.NewScanner(conn)
    for scanner.Scan() {
        var msg Message

		// if reading error, skip
        if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
            continue
        }

		// handle received msg
        response := n.handleMessage(&msg)
        data, _ := json.Marshal(response)
        conn.Write(append(data, '\n'))
    }
}

func (n *Node) handleMessage(msg *Message) *Message {
	// defer to handler based on msg type
    switch msg.Type {
    case MsgWrite:
        return n.handleWrite(msg.Key, msg.Value)
    case MsgReplicate:
        return n.handleReplicate(msg.Entry)
    case MsgGetKey:
        return n.handleGet(msg.Key)
	case MsgHeadHash:
		h, l := n.headHash()
		return &Message{Success: true, Hash: h, Length: l}
	case MsgGetChain:
    	return &Message{Success: true, Entries: n.chain.Entries()}
    default:
        return &Message{Success: false, Error: "unknown message type"}
    }
}

func (n *Node) persistIfConfigured() {
    if n.logFile == "" {
        return
    }
    if err := n.chain.SaveToFile(n.logFile); err != nil {
        fmt.Printf("[%s] warning: could not persist chain: %v\n", n.ID, err)
    }
}

func (n *Node) handleWrite(key, value string) *Message {
    entry := n.chain.Append(key, value)
    n.store.Set(key, value)

    // replicate to all peers
    go n.broadcast(entry)
	n.persistIfConfigured()

    return &Message{Success: true, Entry: entry}
}

func (n *Node) handleReplicate(entry *log.LogEntry) *Message {
    n.mu.RLock()
    trusted := n.trustedKeys
    n.mu.RUnlock()

    if err := n.chain.AcceptEntry(entry, trusted); err != nil {
        return &Message{Success: false, Error: err.Error()}
    }

    n.store.Set(entry.Key, entry.Value)
	n.persistIfConfigured()

    return &Message{Success: true}
}

func (n *Node) handleGet(key string) *Message {
    v, ok := n.store.Get(key)
    if !ok {
        return &Message{Success: false, Error: "key not found"}
    }

    return &Message{Success: true, Value: v}
}

func (n *Node) broadcast(entry *log.LogEntry) {
    n.mu.RLock()
    peers := make(map[string]string)
    for id, addr := range n.peers {
        peers[id] = addr
    }

    n.mu.RUnlock()

    msg := &Message{Type: MsgReplicate, Entry: entry}
    data, _ := json.Marshal(msg)

    for id, addr := range peers {
        conn, err := net.Dial("tcp", addr)
        if err != nil {
            fmt.Printf("[%s] could not reach peer %s: %v\n", n.ID, id, err)
            continue
        }
        conn.Write(append(data, '\n'))
        conn.Close()
    }
}

func (n *Node) Send(addr string, msg *Message) (*Message, error) {
    conn, err := net.Dial("tcp", addr)
    if err != nil {
        return nil, err
    }

    defer conn.Close()

    data, _ := json.Marshal(msg)
    conn.Write(append(data, '\n'))

    scanner := bufio.NewScanner(conn)
    if scanner.Scan() {
        var resp Message
        if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
            return nil, err
        }

        return &resp, nil
    }

    return nil, fmt.Errorf("no response")
}

func (n *Node) PubKeyHex() string {
    return hex.EncodeToString(n.keyPair.Public)
}

func (n *Node) VerifyChain() error {
    n.mu.RLock()
    trusted := n.trustedKeys
    n.mu.RUnlock()

    return n.chain.Verify(trusted)
}

func (n *Node) ChainLen() int {
    return len(n.chain.Entries())
}

func TempKeyPair() (*crypto.KeyPair, error) {
    return crypto.GenerateKeyPair()
}

func (n *Node) headHash() (string, int) {
    entries := n.chain.Entries()
    if len(entries) == 0 {
        return "genesis", 0
    }

    return entries[len(entries)-1].Hash(), len(entries)
}

func (n *Node) CheckDivergence() {
    localHash, localLen := n.headHash()

    n.mu.RLock()
    peers := make(map[string]string)
    for id, addr := range n.peers {
        peers[id] = addr
    }

    n.mu.RUnlock()

    diverged := false
    for peerID, addr := range peers {
        msg := &Message{
            Type:   MsgHeadHash,
            NodeID: n.ID,
            Hash:   localHash,
            Length: localLen,
        }

        resp, err := n.Send(addr, msg)
        if err != nil {
            fmt.Printf("[%s] could not reach %s for divergence check\n", n.ID, peerID)
            continue
        }

        if resp.Length != localLen {
            fmt.Printf("[%s] WARNING: chain length mismatch with %s — local=%d peer=%d\n",
                n.ID, peerID, localLen, resp.Length)
            diverged = true
            continue
        }

        if resp.Hash != localHash {
            fmt.Printf("[%s] CRITICAL: chain divergence detected with %s!\n", n.ID, peerID)
            fmt.Printf("  local head:  %s\n", localHash)
            fmt.Printf("  peer head:   %s\n", resp.Hash)
            diverged = true
        }
    }

    if !diverged {
        fmt.Printf("[%s] divergence check passed — all peers agree\n", n.ID)
    }
}

func (n *Node) WithLogFile(path string) {
    n.logFile = path
}

func (n *Node) LoadLog(path string) error {
    n.logFile = path
    n.mu.RLock()
    trusted := n.trustedKeys
    n.mu.RUnlock()

    if err := n.chain.LoadFromFile(path, trusted); err != nil {
        return err
    }

    // replay the chain into the KV store
    for _, entry := range n.chain.Entries() {
        n.store.Set(entry.Key, entry.Value)
    }

    return nil
}