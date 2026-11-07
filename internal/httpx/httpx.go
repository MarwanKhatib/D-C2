// Package httpx provides a shared HTTP client with sane timeouts.
package httpx

import (
	"net/http"
	"time"
)

// Client is used for Discord attachment transfers. The timeout bounds slow or
// stalled peers while leaving room for multi-megabyte downloads.
var Client = &http.Client{Timeout: 5 * time.Minute}
