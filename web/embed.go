package web

import (
	"embed"
	"io/fs"
)

// DistFS contains the compiled static React 19 web application.
//
//go:embed all:dist
var distFS embed.FS

// GetDistFS returns an fs.FS sub-filesystem rooted at "dist".
func GetDistFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
