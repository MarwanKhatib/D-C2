// Package transfer sends local files to Discord, splitting files that exceed
// the attachment limit into parts with a manifest.
package transfer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bwmarrin/discordgo"
)

// DiscordLimit is the attachment limit used by this client.
const DiscordLimit int64 = 8 * 1024 * 1024

// DefaultChunkSize leaves headroom below DiscordLimit.
const DefaultChunkSize int64 = 7 * 1024 * 1024

// Part describes one split chunk.
type Part struct {
	Index  int    `json:"index"`
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// Manifest describes a split transfer.
type Manifest struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	SHA256    string `json:"sha256"`
	ChunkSize int64  `json:"chunk_size"`
	Chunks    int    `json:"chunks"`
	Parts     []Part `json:"parts"`
}

// NormalizeChunkSize clamps a requested chunk size to a usable range.
func NormalizeChunkSize(chunkSize int64) int64 {
	if chunkSize <= 0 || chunkSize > DiscordLimit {
		return DefaultChunkSize
	}
	return chunkSize
}

// Send uploads path to channelID. Files larger than chunkSize are split into
// parts and accompanied by a manifest; parts and manifest are removed after the
// upload.
func Send(s *discordgo.Session, channelID string, ref *discordgo.MessageReference, path string, chunkSize int64) error {
	chunkSize = NormalizeChunkSize(chunkSize)

	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return fmt.Errorf("%s is a directory", path)
	}

	if fi.Size() <= chunkSize {
		return sendFile(s, channelID, ref, path, "")
	}

	dir, manifest, parts, err := Split(path, chunkSize)
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	manifestPath := filepath.Join(dir, "manifest.json")
	header := fmt.Sprintf(
		"📦 `%s` is %d bytes; split into %d parts (sha256 `%s`). Reassemble with the manifest.",
		manifest.Name, manifest.Size, manifest.Chunks, manifest.SHA256,
	)
	if err := sendFile(s, channelID, ref, manifestPath, header); err != nil {
		return err
	}
	for i, part := range parts {
		label := fmt.Sprintf("part %d/%d · `%s`", i+1, len(parts), filepath.Base(part))
		if err := sendFile(s, channelID, nil, part, label); err != nil {
			return err
		}
	}
	return nil
}

func sendFile(s *discordgo.Session, channelID string, ref *discordgo.MessageReference, path, content string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content:   content,
		Reference: ref,
		Files:     []*discordgo.File{{Name: filepath.Base(path), Reader: f}},
	})
	return err
}

// Split writes chunk files and a manifest into a temporary directory.
func Split(path string, chunkSize int64) (string, Manifest, []string, error) {
	chunkSize = NormalizeChunkSize(chunkSize)

	fi, err := os.Stat(path)
	if err != nil {
		return "", Manifest{}, nil, err
	}

	dir, err := os.MkdirTemp("", "dc2-transfer-")
	if err != nil {
		return "", Manifest{}, nil, err
	}

	whole, err := hashFile(path)
	if err != nil {
		os.RemoveAll(dir)
		return "", Manifest{}, nil, err
	}

	src, err := os.Open(path)
	if err != nil {
		os.RemoveAll(dir)
		return "", Manifest{}, nil, err
	}
	defer src.Close()

	chunks := int((fi.Size() + chunkSize - 1) / chunkSize)
	manifest := Manifest{
		Name:      filepath.Base(path),
		Size:      fi.Size(),
		SHA256:    whole,
		ChunkSize: chunkSize,
		Chunks:    chunks,
		Parts:     make([]Part, 0, chunks),
	}

	var parts []string
	remaining := fi.Size()
	for i := 0; i < chunks; i++ {
		size := chunkSize
		if remaining < size {
			size = remaining
		}
		name := fmt.Sprintf("%s.part%03dof%03d", filepath.Base(path), i+1, chunks)
		partPath := filepath.Join(dir, name)

		partFile, err := os.Create(partPath)
		if err != nil {
			os.RemoveAll(dir)
			return "", Manifest{}, nil, err
		}
		h := sha256.New()
		written, err := io.CopyN(io.MultiWriter(partFile, h), src, size)
		closeErr := partFile.Close()
		if err != nil {
			os.RemoveAll(dir)
			return "", Manifest{}, nil, err
		}
		if closeErr != nil {
			os.RemoveAll(dir)
			return "", Manifest{}, nil, closeErr
		}
		remaining -= written

		manifest.Parts = append(manifest.Parts, Part{
			Index:  i + 1,
			Name:   name,
			Size:   written,
			SHA256: hex.EncodeToString(h.Sum(nil)),
		})
		parts = append(parts, partPath)
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		os.RemoveAll(dir)
		return "", Manifest{}, nil, err
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0o600); err != nil {
		os.RemoveAll(dir)
		return "", Manifest{}, nil, err
	}
	return dir, manifest, parts, nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
