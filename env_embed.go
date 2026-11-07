//go:build envembed

package main

import _ "embed"

// embeddedEnv optionally contains environment configuration baked into the
// binary at build time. Build with `-tags envembed` and a .env file next to
// this source file to enable it.
//
//go:embed .env
var embeddedEnv string
