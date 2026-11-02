package web

import (
	"embed"
	"io/fs"
)

//go:embed static/*
var static embed.FS
var Assets, _ = fs.Sub(static, "static")
