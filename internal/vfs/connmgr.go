package vfs

import (
	"fmt"
	"sync"

	"github.com/feherkaroly/vc/internal/config"
)

// ConnMgr manages active remote filesystem connections.
type ConnMgr struct {
	mu    sync.Mutex
	conns map[string]FileSystem // keyed by server name
}

// NewConnMgr creates a new connection manager.
func NewConnMgr() *ConnMgr {
	return &ConnMgr{
		conns: make(map[string]FileSystem),
	}
}

// Connect returns an existing connection for the given server config, or creates a new one.
func (cm *ConnMgr) Connect(cfg config.ServerConfig) (FileSystem, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if fs, ok := cm.conns[cfg.Name]; ok {
		return fs, nil
	}

	var fs FileSystem
	var err error

	switch cfg.Protocol {
	case "sftp":
		fs, err = NewSFTPFS(cfg)
	case "ftp", "ftps":
		fs, err = NewFTPFS(cfg)
	default:
		return nil, fmt.Errorf("unknown protocol: %s", cfg.Protocol)
	}

	if err != nil {
		return nil, err
	}

	cm.conns[cfg.Name] = fs
	return fs, nil
}

// Disconnect closes the connection for the given server name.
func (cm *ConnMgr) Disconnect(name string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if fs, ok := cm.conns[name]; ok {
		fs.Close()
		delete(cm.conns, name)
	}
}

// IsConnected returns true if the given server name has an active connection.
func (cm *ConnMgr) IsConnected(name string) bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	_, ok := cm.conns[name]
	return ok
}

// DisconnectAll closes all active connections. Call on application exit.
func (cm *ConnMgr) DisconnectAll() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for name, fs := range cm.conns {
		fs.Close()
		delete(cm.conns, name)
	}
}
