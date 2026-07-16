package live

import (
	"embed"
	"io/fs"
)

//go:embed app/*
var appFS embed.FS

//go:embed site/*
var siteFS embed.FS

func AppFS() fs.FS {
	fsys, err := fs.Sub(appFS, "app")
	if err != nil {
		panic(err)
	}
	return fsys
}

// SiteFS is the public landing page served at /. It presents the product and
// links to the app; it holds no board data, so it stays reachable even when the
// instance is gated by basic auth.
func SiteFS() fs.FS {
	fsys, err := fs.Sub(siteFS, "site")
	if err != nil {
		panic(err)
	}
	return fsys
}
