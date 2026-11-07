//go:build ffmpegembed && windows
// +build ffmpegembed,windows

package media

import _ "embed"

//go:embed ffmpeg/windows/ffmpeg.exe
var embeddedFFmpegWin []byte
