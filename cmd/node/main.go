package main

import (
	"bufio"
	"crypto/ed25519"
	"encoding/hex"
	"flag"
	"fmt"
	"kvstore/internal/config"
	"kvstore/internal/crypto"
	"kvstore/internal/network"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
    id         := flag.String("id", "", "node ID (must match config)")
    configPath := flag.String("config", "config.json", "path to config file")
    flag.Parse()

    if *id == "" {
        fmt.Println("usage: node -id <id> [-config config.json]")
        os.Exit(1)
    }

    cfg, err := config.Load(*configPath)
    if err != nil {
        fmt.Println("failed to load config:", err)
        os.Exit(1)
    }

    self := cfg.FindSelf(*id)
    if self == nil {
        fmt.Printf("node %q not found in config\n", *id)
        os.Exit(1)
    }

    // Load or generate keypair
    kp, err := crypto.LoadFromFile(self.KeyFile)
    if err != nil {
        fmt.Printf("[%s] generating new keypair -> %s\n", *id, self.KeyFile)
        kp, err = crypto.GenerateKeyPair()
        if err != nil {
            fmt.Println("failed to generate keypair:", err)
            os.Exit(1)
        }

        kp.SaveToFile(self.KeyFile)
    }

    pubHex := hex.EncodeToString(kp.Public)

    // If this node's pubkey isn't in the config yet, write it in
    if self.PubKey == "" {
        self.PubKey = pubHex
        if err := cfg.Save(*configPath); err != nil {
            fmt.Println("warning: could not save pubkey to config:", err)
        } else {
            fmt.Printf("[%s] saved public key to config\n", *id)
        }
    }

    fmt.Printf("[%s] public key: %s\n", *id, pubHex)

    node := network.NewNode(*id, self.Addr, kp)

    // Register all peers that have a pubkey in the config
    registered := 0
    for _, peer := range cfg.Peers(*id) {
        if peer.PubKey == "" {
            fmt.Printf("[%s] skipping peer %s — no pubkey in config yet\n", *id, peer.ID)
            continue
        }
        pubBytes, err := hex.DecodeString(peer.PubKey)
        if err != nil {
            fmt.Printf("[%s] invalid pubkey for peer %s\n", *id, peer.ID)
            continue
        }
        node.AddPeer(peer.ID, peer.Addr, ed25519.PublicKey(pubBytes))
        fmt.Printf("[%s] registered peer %s\n", *id, peer.ID)
        registered++
    }

    fmt.Printf("[%s] %d peers registered\n", *id, registered)

	logFile := *id + ".log"
	if err := node.LoadLog(logFile); err != nil {
		fmt.Println("warning: could not load log:", err)
	} else {
		fmt.Printf("[%s] chain replayed — %d entries\n", *id, node.ChainLen())
	}

    if err := node.Start(); err != nil {
        fmt.Println("failed to start node:", err)
        os.Exit(1)
    }

    fmt.Println("commands: verify | divergence | quit")
    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        parts := strings.Fields(line)
        if len(parts) == 0 {
            continue
        }

        switch parts[0] {
        case "verify":
            if err := node.VerifyChain(); err != nil {
                fmt.Println("CHAIN INVALID:", err)
            } else {
                fmt.Println("chain OK —", node.ChainLen(), "entries")
            }
        case "quit":
            os.Exit(0)
		case "divergence":
    		node.CheckDivergence()
        default:
            fmt.Println("unknown command")
        }
    }

    sig := make(chan os.Signal, 1)
    signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
    <-sig
}