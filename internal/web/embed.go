// Package web holds the embedded mobile web app.
package web

import "embed"

//go:embed static
var Files embed.FS
