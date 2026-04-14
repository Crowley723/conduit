package main

import (
	"flag"
	"fmt"
	"github.com/Crowley723/conduit/internal/config"
	"github.com/Crowley723/conduit/internal/server"
	"github.com/Crowley723/conduit/internal/version"
	"log"
	"os"
)

func main() {
	configPath := flag.String("config", "", "path to config file")
	flag.StringVar(configPath, "c", "", "path to config file (shorthand)")

	versionFlag := flag.Bool("version", false, "print version information")
	flag.BoolVar(versionFlag, "v", false, "print version information (shorthand)")

	flag.Parse()

	if *versionFlag {
		fmt.Printf("Version: %s\n", version.GetVersion())
		fmt.Printf("Git Commit: %s\n", version.GetGitCommit())
		fmt.Printf("Build Time: %s\n", version.GetBuildTime())
		os.Exit(0)
	}

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	s, err := server.New(cfg)
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	err = s.Start()
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
