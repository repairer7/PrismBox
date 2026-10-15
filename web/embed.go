package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist/*
var embeddedFS embed.FS

func GetFileSystem() http.FileSystem {
	sub, _ := fs.Sub(embeddedFS, "dist")
	return http.FS(sub)
}
