package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type PanelConfig struct {
	Mode     int    `json:"mode"`
	SortMode int    `json:"sort_mode"`
	Path     string `json:"path,omitempty"`
}

type ServerConfig struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"` // "sftp", "ftp", "ftps"
	Host     string `json:"host"`
	Port     int    `json:"port,omitempty"` // 0 = default (22/21)
	User     string `json:"user"`
	Password string `json:"password,omitempty"`
	KeyPath  string `json:"key_path,omitempty"`
}

type Config struct {
	LeftPanel   PanelConfig       `json:"left_panel"`
	RightPanel  PanelConfig       `json:"right_panel"`
	ActivePanel int               `json:"active_panel"`
	Servers     []ServerConfig    `json:"servers,omitempty"`
	QuickPaths  map[string]string `json:"quick_paths,omitempty"`
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
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return
	}
	// Use 0600 if any server has a password
	perm := os.FileMode(0644)
	for _, s := range c.Servers {
		if s.Password != "" {
			perm = 0600
			break
		}
	}
	os.WriteFile(p, data, perm)
}
