# Distributed Key-Value Store with Tamper-Evident Replication

A distributed key-value store written in Go where every write is
cryptographically signed and hash-chained across all nodes. A compromised
or malicious node cannot silently alter the log without detection.

## Architecture

The system consists of any number of nodes, each running as an independent
process. Nodes communicate over raw TCP using newline-delimited JSON messages.
There is no central coordinator — any node accepts writes and replicates them
to all peers.

Each node maintains three things:

- **A KV store** — an in-memory map reflecting the latest value for each key
- **An append-only log** — every write is recorded as a `LogEntry`, persisted
  to disk as newline-delimited JSON
- **A chain** — each log entry contains the SHA-256 hash of the previous entry,
  forming a tamper-evident chain from the genesis entry forward
```
┌─────────────────────────────────────────────────────┐
│  Client                                             │
│  ./client -op set -key foo -val bar                 │
└───────────────────┬─────────────────────────────────┘
                    │ TCP (JSON)
          ┌─────────-──────────┐
          │  Node 1 (primary)  │
          │  - signs entry     │
          │  - appends to log  │
          │  - updates store   │
          │  - broadcasts      │
          └──────┬──────┬──────┘
         TCP     │      │     TCP
      ┌──────────-─┐  ┌─-──────────┐
      │   Node 2   │  │   Node 3   │
      │ - verifies │  │ - verifies │
      │ - appends  │  │ - appends  │
      └────────────┘  └────────────┘
```

## Security Model

### Threat model

The system is designed to detect a Byzantine node — one that has been
compromised and attempts to silently alter the log. The attacker is assumed
to have full control of one node, including its network connection, but
does not have access to other nodes' private keys.

The system does **not** protect against:

- A compromised node that simply drops writes (availability attack)
- An attacker who has stolen a node's private key
- A global network partition where nodes cannot compare chain heads
- Sybil attacks (the trusted key registry is static)

### How tamper detection works

Every `LogEntry` contains:
```
Index     — position in the log
Key       — the key being written  
Value     — the value being written
NodeID    — which node authored this entry
Timestamp — Unix nanoseconds
PrevHash  — SHA-256 hash of the previous entry
Signature — Ed25519 signature over all of the above
```

Two independent mechanisms protect the log:

**1. Signature verification**
Each entry is signed by the writing node using its Ed25519 private key.
Peers verify the signature against a static registry of trusted public keys
before accepting any entry. An entry with an unknown signing key or an
invalid signature is rejected outright.

**2. Hash chaining**
Each entry commits to the hash of the entry before it. Altering any entry
changes its hash, which invalidates the `PrevHash` field of every subsequent
entry. An attacker cannot alter a historical entry without rewriting the
entire chain from that point forward and cannot do so without a valid
signing key.

### Key management

Each node generates an Ed25519 keypair on first startup and persists the
private key to disk (chmod 0600). Public keys are distributed via
`config.json`, which acts as a static PKI registry. All nodes share the
same config file. Adding a node requires generating its key and publishing
its public key to all peers before it can participate in replication.

## Running a cluster

**1. Build**
```bash
go build ./cmd/node/
go build ./cmd/client/
go build ./cmd/byzantine/
```

**2. Configure**

Create `config.json`:
```json
{
  "nodes": [
    {"id": "node1", "addr": "localhost:9001", "key": "node1.key", "pub_key": ""},
    {"id": "node2", "addr": "localhost:9002", "key": "node2.key", "pub_key": ""},
    {"id": "node3", "addr": "localhost:9003", "key": "node3.key", "pub_key": ""}
  ]
}
```

**3. Bootstrap (first run only)**

Start each node once to generate keypairs and populate `config.json`:
```bash
./node -id node1
./node -id node2
./node -id node3
```

After all three have written their public keys to `config.json`, kill them (or quit)
and restart. From this point on the cluster self-assembles on startup.

**4. Normal startup**
```bash
./node -id node1 &
./node -id node2 &
./node -id node3 &
```

Each node replays its persisted log and registers peers automatically.

**5. Write and read**
```bash
./client -addr localhost:9001 -op set -key user -val root
./client -addr localhost:9003 -op get -key user   # reads from a different node
```

**6. Verify chain integrity**

Type `verify` in any node terminal: (where x is the number of made entries)
```
chain OK — x entries
```

**7. Check for divergence across the cluster**

Type `divergence` in any node terminal:
```
[node1] divergence check passed — all peers agree
```

## Byzantine attack simulation
```bash
./byzantine localhost:9001
```

Each attack is blocked by a different layer of the security model:

| Attack | Mechanism | Reason blocked |
|---|---|---|
| Value tampering | Signature verification | Payload changed, signature invalid |
| Entry replay | Hash chaining | PrevHash does not match expected |
| Forged entry | PKI registry | Signing key not in trusted set |

## Project structure
```
kvstore/
├── cmd/
│   ├── node/        — node binary
│   ├── client/      — CLI client
│   └── byzantine/   — attack simulator
├── internal/
│   ├── config/      — config loading and PKI registry
│   ├── crypto/      — Ed25519 key generation, signing, verification
│   ├── log/         — LogEntry, Chain, persistence
│   ├── network/     — Node, TCP message handling, replication
│   └── store/       — in-memory KV store
├── config.json      — cluster topology and trusted public keys
└── *.key            — per-node private keys (not committed)
```
