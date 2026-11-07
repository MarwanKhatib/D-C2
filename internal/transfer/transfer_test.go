package transfer

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestSplitReassembles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.bin")
	original := make([]byte, 2*1024*1024+512*1024)
	for i := range original {
		original[i] = byte(i * 31)
	}
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}

	outDir, manifest, parts, err := Split(path, 1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outDir)

	if manifest.Chunks != 3 || len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d/%d", manifest.Chunks, len(parts))
	}
	if manifest.Size != int64(len(original)) {
		t.Fatalf("manifest size = %d", manifest.Size)
	}

	var joined []byte
	for i, part := range parts {
		data, err := os.ReadFile(part)
		if err != nil {
			t.Fatal(err)
		}
		joined = append(joined, data...)
		sum := sha256.Sum256(data)
		if got := hex.EncodeToString(sum[:]); got != manifest.Parts[i].SHA256 {
			t.Fatalf("part %d hash mismatch", i+1)
		}
	}
	if !bytes.Equal(joined, original) {
		t.Fatal("reassembled data differs from original")
	}
	sum := sha256.Sum256(original)
	if manifest.SHA256 != hex.EncodeToString(sum[:]) {
		t.Fatal("manifest whole-file hash mismatch")
	}
}

func TestNormalizeChunkSize(t *testing.T) {
	if got := NormalizeChunkSize(0); got != DefaultChunkSize {
		t.Fatalf("zero -> %d", got)
	}
	if got := NormalizeChunkSize(DiscordLimit + 1); got != DefaultChunkSize {
		t.Fatalf("oversize -> %d", got)
	}
	if got := NormalizeChunkSize(1024); got != 1024 {
		t.Fatalf("valid -> %d", got)
	}
}
