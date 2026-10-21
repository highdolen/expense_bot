package rest

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed web/*
var webFiles embed.FS

func NewWebHandler() http.Handler {
	files, err := fs.Sub(webFiles, "web")
	if err != nil {
		panic(err)
	}

	return http.FileServer(http.FS(files))
}
