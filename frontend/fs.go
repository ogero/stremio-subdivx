package frontend

import "embed"

// Dist embeds the addon frontend
//
//go:embed dist
var Dist embed.FS
