// Package cartrak carries files from the repository root that are compiled into the binary.
package cartrak

import _ "embed"

// Changelog is CHANGELOG.md, shown in the app's "What's new" page.
//
//go:embed CHANGELOG.md
var Changelog string
