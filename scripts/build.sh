#!/usr/bin/env bash
# Build release artifacts ("payload generator").
#
# Usage:
#   scripts/build.sh              # linux + windows + darwin
#   scripts/build.sh linux        # one target
#   EMBED_ENV=1 scripts/build.sh  # bake .env into the binary
#   EMBED_FFMPEG=1 scripts/build.sh windows
set -euo pipefail
cd "$(dirname "$0")/.."

GO="${GO:-go}"
OUT="${OUT:-dist}"
TARGET="${1:-all}"

tags=()
if [[ "${EMBED_ENV:-0}" == "1" ]]; then
  [[ -f .env ]] || { echo "EMBED_ENV=1 requires .env in the repository root" >&2; exit 1; }
  tags+=("envembed")
fi
if [[ "${EMBED_FFMPEG:-0}" == "1" ]]; then
  [[ -f internal/media/ffmpeg/windows/ffmpeg.exe ]] || {
    echo "EMBED_FFMPEG=1 requires internal/media/ffmpeg/windows/ffmpeg.exe" >&2
    exit 1
  }
  tags+=("ffmpegembed")
fi

tag_flag=()
if [[ ${#tags[@]} -gt 0 ]]; then
  tag_flag=(-tags "$(IFS=,; echo "${tags[*]}")")
fi

mkdir -p "$OUT"
rm -f "$OUT"/SHA256SUMS

build() {
  local goos="$1" goarch="$2" name="$3"
  echo "[build] ${goos}/${goarch} -> ${OUT}/${name}"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    "$GO" build -trimpath "${tag_flag[@]}" -ldflags "-s -w" -o "${OUT}/${name}" .
  ( cd "$OUT" && sha256sum "$name" >> SHA256SUMS )
}

case "$TARGET" in
  all)
    build linux amd64 dc2-linux-amd64
    build linux arm64 dc2-linux-arm64
    build windows amd64 dc2-windows-amd64.exe
    build darwin arm64 dc2-darwin-arm64
    ;;
  linux)   build linux amd64 dc2-linux-amd64 ;;
  windows) build windows amd64 dc2-windows-amd64.exe ;;
  darwin)  build darwin arm64 dc2-darwin-arm64 ;;
  *) echo "unknown target: $TARGET (linux|windows|darwin|all)" >&2; exit 2 ;;
esac

echo "[build] done"
cat "$OUT/SHA256SUMS"
