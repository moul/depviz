package live

import (
	"embed"
	"io/fs"
)

//go:embed app/*
var appFS embed.FS

func AppFS() fs.FS {
	fsys, err := fs.Sub(appFS, "app")
	if err != nil {
		panic(err)
	}
	return fsys
}
