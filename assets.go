package main

import "embed"

// Embedded browser assets keep the source readable and avoid a large inline script.
//
//go:embed assets/icon.png
var appIconPNG []byte

//go:embed assets/audio-artwork/*.png
var audioArtworkFS embed.FS

//go:embed web/index.html
var appHTML string

//go:embed web/style.css
var appCSS []byte

//go:embed web/app.js
var appJS []byte

//go:embed web/i18n.js
var appI18nJS []byte

//go:embed frontend/shared/gba-text.js
var gbaTextJS []byte

//go:embed frontend/shared/menu-themes.js
var menuThemesJS []byte

//go:embed frontend/shared/title-cards.js
var titleCardsJS []byte

//go:embed locales/*
var localeFS embed.FS
