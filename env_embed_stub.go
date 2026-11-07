//go:build !envembed

package main

// embeddedEnv is empty for default builds. Use `-tags envembed` to bake a
// local .env file into the binary.
var embeddedEnv string
