package config

import (
	"encoding/json"
	"os"
)

type ClientConfig struct {
	SignalingUrl  string `json:"signalingUrl"`
	SignalingPort string `json:"signalingPort"`
	DisplayName   string `json:"displayName"`
	IdleTimeout   int16  `json:"idleTimeout"`
}

func LoadFromJson(path string) (*ClientConfig, error) {
	buff, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config ClientConfig

	err = json.Unmarshal(buff, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
