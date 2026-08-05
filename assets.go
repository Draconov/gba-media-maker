package main

import _ "embed"

// Embedded browser assets keep the source readable and avoid a large inline script.
//
//go:embed assets/app_icon.png
var appIconPNG []byte

//go:embed web/index.html
var appHTML string

//go:embed web/style.css
var appCSS []byte

//go:embed web/app.js
var appJS []byte

//go:embed web/menu-themes.js
var menuThemesJS []byte
