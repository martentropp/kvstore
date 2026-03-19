package network

import "kvstore/internal/log"

type MessageType string

const (
    MsgWrite     MessageType = "WRITE"
    MsgReplicate MessageType = "REPLICATE"
    MsgGetKey    MessageType = "GET"
	MsgGetChain  MessageType = "GET_CHAIN"
    MsgResponse  MessageType = "RESPONSE"
    MsgJoin      MessageType = "JOIN"
	MsgHeadHash  MessageType = "HEAD_HASH"
	MsgHeadReply MessageType = "HEAD_REPLY"
)

type Message struct {
    Type    MessageType     `json:"type"`
    Entry   *log.LogEntry   `json:"entry,omitempty"`
    Entries []*log.LogEntry `json:"entries,omitempty"`
    Key     string          `json:"key,omitempty"`
    Value   string          `json:"value,omitempty"`
    NodeID  string          `json:"node_id,omitempty"`
    PubKey  string          `json:"pub_key,omitempty"`
    Hash    string          `json:"hash,omitempty"`
    Length  int             `json:"length,omitempty"`
    Success bool            `json:"success,omitempty"`
    Error   string          `json:"error,omitempty"`
}