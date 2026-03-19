package log

import (
	"bufio"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"os"
)

func (c *Chain) SaveToFile(path string) error {
    c.mu.RLock()
    defer c.mu.RUnlock()

    f, err := os.Create(path)
    if err != nil {
        return err
    }

    defer f.Close()

    w := bufio.NewWriter(f)
    for _, entry := range c.entries {
        data, err := json.Marshal(entry)
        if err != nil {
            return err
        }
        fmt.Fprintf(w, "%s\n", data)
    }

    return w.Flush()
}

func (c *Chain) LoadFromFile(path string, trustedKeys map[string]ed25519.PublicKey) error {
    f, err := os.Open(path)
    if os.IsNotExist(err) {
        return nil // fresh start, not an error
    }

    if err != nil {
        return err
    }
	
    defer f.Close()

    c.mu.Lock()
    defer c.mu.Unlock()

    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        var entry LogEntry
        if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
            return fmt.Errorf("corrupt log entry: %w", err)
        }
        c.entries = append(c.entries, &entry)
    }

    fmt.Printf("loaded %d entries from %s\n", len(c.entries), path)
    return scanner.Err()
}