package main

import (
	_ "embed"
	"strings"
)

// rawAppVersion is the single source of truth for the application release
// version. Update VERSION when preparing a release; desktop and website build
// tooling consume the same file.
//
//go:embed VERSION
var rawAppVersion string

var appVersion = strings.TrimSpace(rawAppVersion)
