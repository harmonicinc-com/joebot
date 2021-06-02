// +build !dev

package main

import (
	"embed"
	"io/fs"
)

//go:embed web_portal_html/*
var webPortalAssets embed.FS

func WebPortalAssetsFS() fs.FS {
	assetsFs, _ := fs.Sub(webPortalAssets, "web_portal_html")
	return assetsFs
}
