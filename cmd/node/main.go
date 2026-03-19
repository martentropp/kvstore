package main

import (
	"bufio"
	"crypto/ed25519"
	"encoding/hex"
	"flag"
	"fmt"
	"kvstore/internal/crypto"
	"kvstore/internal/network"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	// command line parsing
    id   := flag.String("id", "", "node ID (e.g. node1)")
    addr := flag.String("addr", "", "address to listen on (e.g. localhost:9001)")
    keyFile := flag.String("key", "", "path to key file (created if missing)")
    flag.Parse()

	// verify correct usage
    if *id == "" || *addr == "" || *keyFile == "" {
        fmt.Println("usage: node -id <id> -addr <addr> -key <keyfile>")
        os.Exit(1)
    }

    // load or generate keypair
    kp, err := crypto.LoadFromFile(*keyFile)
    if err != nil {
        fmt.Printf("[%s] generating new keypair -> %s\n", *id, *keyFile)
        kp, err = crypto.GenerateKeyPair()
        if err != nil {
            fmt.Println("failed to generate keypair:", err)
            os.Exit(1)
        }

        kp.SaveToFile(*keyFile)
    }

    fmt.Printf("[%s] public key: %s\n", *id, hex.EncodeToString(kp.Public))

    node := network.NewNode(*id, *addr, kp)

    if err := node.Start(); err != nil {
        fmt.Println("failed to start node:", err)
        os.Exit(1)
    }

    // interactive peer registration + commands
    fmt.Println("commands: peer <id> <addr> <pubkey> | verify | quit")
    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        parts := strings.Fields(line)
        if len(parts) == 0 {
            continue
        }

        switch parts[0] {
        case "peer":
            if len(parts) != 4 {
                fmt.Println("usage: peer <id> <addr> <pubkey>")
                continue
            }
            pubBytes, err := hex.DecodeString(parts[3])
            if err != nil {
                fmt.Println("invalid pubkey hex")
                continue
            }
            node.AddPeer(parts[1], parts[2], ed25519.PublicKey(pubBytes))
            fmt.Printf("added peer %s at %s\n", parts[1], parts[2])

        case "verify":
            if err := node.VerifyChain(); err != nil {
                fmt.Println("CHAIN INVALID:", err)
            } else {
                fmt.Println("chain OK —", node.ChainLen(), "entries")
            }

        case "quit":
            os.Exit(0)

        default:
            fmt.Println("unknown command")
        }
    }

    // handle Ctrl + C
    sig := make(chan os.Signal, 1)
    signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
    <-sig
}