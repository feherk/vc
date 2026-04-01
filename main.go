package main

import (
	"flag"
	"fmt"
	"os"

	application "github.com/feherkaroly/vc/internal/app"
	"github.com/feherkaroly/vc/internal/config"
)

var Version = "3.0.2"

func main() {
	showVersion := flag.Bool("version", false, "Show version")
	leftDir := flag.String("left", ".", "Left panel directory")
	rightDir := flag.String("right", ".", "Right panel directory")
	flag.Parse()

	if *showVersion {
		fmt.Printf("vc %s\n", Version)
		return
	}

	// Resolve CWD
	wd, _ := os.Getwd()
	if wd == "" {
		wd = "."
	}

	// Use saved paths if no explicit arguments given
	if *leftDir == "." && *rightDir == "." {
		cfg := config.Load()
		// Active panel gets CWD, inactive panel gets saved path
		if cfg.ActivePanel == 0 {
			// Left is active → left=CWD, right=saved
			*leftDir = wd
			if cfg.RightPanel.Path != "" {
				if info, err := os.Stat(cfg.RightPanel.Path); err == nil && info.IsDir() {
					*rightDir = cfg.RightPanel.Path
				}
			}
		} else {
			// Right is active → right=CWD, left=saved
			*rightDir = wd
			if cfg.LeftPanel.Path != "" {
				if info, err := os.Stat(cfg.LeftPanel.Path); err == nil && info.IsDir() {
					*leftDir = cfg.LeftPanel.Path
				}
			}
		}
	}

	// Resolve any remaining relative paths
	if *leftDir == "." {
		*leftDir = wd
	}
	if *rightDir == "." {
		*rightDir = wd
	}

	application.Version = Version
	app := application.New(*leftDir, *rightDir)
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
