# Build artifacts (payload generator)

`scripts/build.sh` is the generator: it produces per-OS binaries and a
`SHA256SUMS` file.

## 1. Targets

| Argument | Output |
|---|---|
| `linux` | `dist/dc2-linux-amd64` |
| `windows` | `dist/dc2-windows-amd64.exe` |
| `darwin` | `dist/dc2-darwin-arm64` |
| `all` (default) | linux amd64/arm64, windows amd64, darwin arm64 |

```bash
scripts/build.sh
scripts/build.sh windows
OUT=/srv/payloads scripts/build.sh linux
```

Binaries are built with `CGO_ENABLED=0`, `-trimpath`, and `-ldflags "-s -w"`,
so they are static and reasonably small.

## 2. Build tags

| Variable | Tag | Effect |
|---|---|---|
| `EMBED_ENV=1` | `envembed` | Bakes `.env` into the binary. Requires `.env` at build time. |
| `EMBED_FFMPEG=1` | `ffmpegembed` | Embeds `internal/media/ffmpeg/windows/ffmpeg.exe` (Windows). |

```bash
EMBED_ENV=1 scripts/build.sh windows
EMBED_ENV=1 EMBED_FFMPEG=1 scripts/build.sh windows
```

**Embedding the token makes the artifact a secret.** Anyone with the binary can
extract it. Prefer a sidecar `.env` for anything shared, and rotate the token if
an embedded artifact is lost.

## 3. Verify an artifact

```bash
sha256sum -c dist/SHA256SUMS
file dist/dc2-linux-amd64
```

On Windows:

```powershell
Get-FileHash dist\dc2-windows-amd64.exe -Algorithm SHA256
```

## 4. Reproducibility

- Pin the Go toolchain version used for a release.
- Commit `go.mod` and `go.sum` (already tracked).
- Add `-buildvcs=false` if you need identical hashes across checkouts.

## 5. Manual build (no script)

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -ldflags "-s -w" -o dc2-client .
```

Cross-compilation works from any OS. `darwin` requires `CGO_ENABLED=0`; Windows
and Linux build either way.
