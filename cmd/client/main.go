package main

import (
	"flag"
	"fmt"
	"kvstore/internal/network"
	"os"
)

func main() {
	// command line parsing
    addr := flag.String("addr", "localhost:9001", "node address")
    op   := flag.String("op", "", "operation: get or set")
    key  := flag.String("key", "", "key")
    val  := flag.String("val", "", "value (for set)")
    flag.Parse()

	// verify correctness
    if *op == "" || *key == "" {
        fmt.Println("usage: client -addr <addr> -op <get|set> -key <key> [-val <value>]")
        os.Exit(1)
    }

    // temporary node just for sending
    kp, _ := network.TempKeyPair()
    sender := network.NewNode("client", "localhost:0", kp)

    var msg *network.Message
    switch *op {
    case "set":
        if *val == "" {
            fmt.Println("-val required for set")
            os.Exit(1)
        }
        msg = &network.Message{Type: network.MsgWrite, Key: *key, Value: *val}
    case "get":
        msg = &network.Message{Type: network.MsgGetKey, Key: *key}
    default:
        fmt.Println("op must be get or set")
        os.Exit(1)
    }

    resp, err := sender.Send(*addr, msg)
    if err != nil {
        fmt.Println("error:", err)
        os.Exit(1)
    }

    if !resp.Success {
        fmt.Println("error:", resp.Error)
        os.Exit(1)
    }

    switch *op {
    case "set":
        fmt.Printf("ok — entry %d\n", resp.Entry.Index)
    case "get":
        fmt.Println(resp.Value)
    }
}