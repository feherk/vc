package main

import (
	"flag"
	"fmt"
	"os"

	application "github.com/feherkaroly/vc/internal/app"
)

var Version = "1.0.0"

func main() {
	showVersion := flag.Bool("version", false, "Show version")
	leftDir := flag.String("left", ".", "Left panel directory")
	rightDir := flag.String("right", ".", "Right panel directory")
	flag.Parse()

	if *showVersion {
		fmt.Printf("vc %s\n", Version)
		return
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
