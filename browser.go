package monitoring

import "embed"

// browserFS contains the compiled frontend dashboard assets.
// They are embedded at build time so the dashboard works without
// any external filesystem dependency.
//
//go:embed all:browser
var browserFS embed.FS
