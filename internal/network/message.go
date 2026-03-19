package network

import "kvstore/internal/log"

type MessageType string

const (
    MsgWrite     MessageType = "WRITE"
    MsgReplicate MessageType = "REPLICATE"
    MsgGetKey    MessageType = "GET"
    MsgResponse  MessageType = "RESPONSE"
    MsgJoin      MessageType = "JOIN"
)

type Message struct {
    Type    MessageType  `json:"type"`
    Entry  *log.LogEntry `json:"entry,omitempty"`
    Key     string       `json:"key,omitempty"`
    Value   string       `json:"value,omitempty"`
    NodeID  string       `json:"node_id,omitempty"`
    PubKey  string       `json:"pub_key,omitempty"`
    Success bool         `json:"success,omitempty"`
    Error   string       `json:"error,omitempty"`
}