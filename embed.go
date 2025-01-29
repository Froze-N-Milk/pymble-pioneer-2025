package root

import(
	"embed"
)

//go:embed templates/*
var Templates embed.FS

//go:embed content/*
var JSLib embed.FS
