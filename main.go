package main

import (
	"flag"
	"fmt"
	"os"

	application "github.com/feherkaroly/vc/internal/app"
	"github.com/feherkaroly/vc/internal/config"
)

var Version = "2.6.6"

func main() {
	showVersion := flag.Bool("version", false, "Show version")
	leftDir := flag.String("left", ".", "Left panel directory")
	rightDir := flag.String("right", ".", "Right panel directory")
	flag.Parse()

	if *showVersion {
		fmt.Printf("vc %s\n", Version)
		return
	}

	// Use saved paths if no explicit arguments given
	if *leftDir == "." && *rightDir == "." {
		cfg := config.Load()
		if cfg.LeftPanel.Path != "" {
			if info, err := os.Stat(cfg.LeftPanel.Path); err == nil && info.IsDir() {
				*leftDir = cfg.LeftPanel.Path
			}
		}
		if cfg.RightPanel.Path != "" {
			if info, err := os.Stat(cfg.RightPanel.Path); err == nil && info.IsDir() {
				*rightDir = cfg.RightPanel.Path
			}
		}
	}

	// Resolve relative paths
	if *leftDir == "." {
		if wd, err := os.Getwd(); err == nil {
			*leftDir = wd
		}
	}
	if *rightDir == "." {
		if wd, err := os.Getwd(); err == nil {
			*rightDir = wd
		}
	}

	application.Version = Version
	app := application.New(*leftDir, *rightDir)
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
