package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type PanelConfig struct {
	Mode     int `json:"mode"`
	SortMode int `json:"sort_mode"`
}

type Config struct {
	LeftPanel  PanelConfig `json:"left_panel"`
	RightPanel PanelConfig `json:"right_panel"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "vc", "config.json")
}

func Load() *Config {
	c := &Config{}
	data, err := os.ReadFile(configPath())
	if err != nil {
		return c
	}
	json.Unmarshal(data, c)
	return c
}

func Save(c *Config) {
	p := configPath()
	os.MkdirAll(filepath.Dir(p), 0755)
	data, err := json.Marshal(c)
	if err != nil {
		return
	}
	os.WriteFile(p, data, 0644)
}
