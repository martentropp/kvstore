package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"kvstore/internal/log"
	"kvstore/internal/network"
	"net"
	"os"
)

// fetchChain connects to a node and asks for its full chain
func fetchChain(addr string) ([]*log.LogEntry, error) {
    conn, err := net.Dial("tcp", addr)
    if err != nil {
        return nil, err
    }
    defer conn.Close()

    msg := &network.Message{Type: network.MsgGetChain}
    data, _ := json.Marshal(msg)
    conn.Write(append(data, '\n'))

    scanner := bufio.NewScanner(conn)
    if scanner.Scan() {
        var resp network.Message
        if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
            return nil, err
        }

        return resp.Entries, nil
    }

    return nil, fmt.Errorf("no response")
}

// injectEntry tries to push a tampered entry directly to a node
func injectEntry(addr string, entry *log.LogEntry) error {
    conn, err := net.Dial("tcp", addr)
    if err != nil {
        return err
    }
    defer conn.Close()

    msg := &network.Message{Type: network.MsgReplicate, Entry: entry}
    data, _ := json.Marshal(msg)
    conn.Write(append(data, '\n'))

    scanner := bufio.NewScanner(conn)
    if scanner.Scan() {
        var resp network.Message
        json.Unmarshal(scanner.Bytes(), &resp)
        if !resp.Success {
            return fmt.Errorf("node rejected entry: %s", resp.Error)
        }
    }
    return nil
}

func main() {
	// defaults to localhost:9001
    target := "localhost:9001"
    if len(os.Args) > 1 {
        target = os.Args[1]
    }

    fmt.Printf("Byzantine Attack Simulation against %s \n\n", target)

    // tamper with a value, keep everything else intact
    fmt.Println("[ Attack 1 ] Fetching chain...")
    entries, err := fetchChain(target)
    if err != nil {
        fmt.Println("failed to fetch chain:", err)
        os.Exit(1)
    }

    fmt.Printf("fetched %d entries\n", len(entries))

    if len(entries) == 0 {
        fmt.Println("chain is empty — write some entries first")
        os.Exit(1)
    }

    // clone the first entry and tamper with it
    tampered := *entries[0]
    original := tampered.Value
    tampered.Value = "TAMPERED"
    fmt.Printf("tampering entry 0: %q -> %q\n", original, tampered.Value)
    fmt.Printf("original hash would be:  %s\n", entries[0].Hash())
    fmt.Printf("tampered hash would be:  %s\n", tampered.Hash())

    err = injectEntry(target, &tampered)
    if err != nil {
        fmt.Printf("Attack 1 blocked: %v\n\n", err)
    } else {
        fmt.Println("Attack 1 succeeded — node accepted tampered entry")
    }

    // replay a valid old entry at a new index
    fmt.Println("[ Attack 2 ] Replaying a valid old entry at wrong index...")
    replayed := *entries[0]
    replayed.Index = uint64(len(entries)) // pretend it's a new entry

    // signature is still valid for the original content, but PrevHash won't match
    err = injectEntry(target, &replayed)
    if err != nil {
        fmt.Printf("Attack 2 blocked: %v\n\n", err)
    } else {
        fmt.Println("Attack 2 succeeded — node accepted replayed entry")
    }

    // forge an entry with a different key
    fmt.Println("[ Attack 3 ] Injecting a forged entry signed by an unknown key...")
    kp, _ := network.TempKeyPair()
    forged := &log.LogEntry{
        Index:    uint64(len(entries)),
        Key:      "admin",
        Value:    "hacked",
        NodeID:   "evil-node",
        Timestamp: 9999999999,
        PrevHash: entries[len(entries)-1].Hash(),
    }
	
    forged.Sign(kp)
    fmt.Printf("forged entry signed with unknown key: %s\n", hex.EncodeToString(kp.Public)[:16]+"...")
    err = injectEntry(target, forged)
    if err != nil {
        fmt.Printf("Attack 3 blocked: %v\n\n", err)
    } else {
        fmt.Println("Attack 3 succeeded — node accepted forged entry")
    }

    fmt.Println("Simulation complete")
}