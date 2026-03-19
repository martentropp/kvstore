package config

import (
	"encoding/json"
	"os"
)

type NodeConfig struct {
    ID     string `json:"id"`
    Addr   string `json:"addr"`
    KeyFile string `json:"key"`
    PubKey string `json:"pub_key"` // hex, populated after first run
}

type Config struct {
    Nodes []*NodeConfig `json:"nodes"`
}

func Load(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var cfg Config
	
    return &cfg, json.Unmarshal(data, &cfg)
}

func (c *Config) Save(path string) error {
    data, err := json.MarshalIndent(c, "", "  ")
    if err != nil {
        return err
    }

    return os.WriteFile(path, data, 0644)
}

func (c *Config) FindSelf(id string) *NodeConfig {
    for _, n := range c.Nodes {
        if n.ID == id {
            return n
        }
    }

    return nil
}

func (c *Config) Peers(selfID string) []*NodeConfig {
    peers := make([]*NodeConfig, 0)
    for _, n := range c.Nodes {
        if n.ID != selfID {
            peers = append(peers, n)
        }
    }

    return peers
}