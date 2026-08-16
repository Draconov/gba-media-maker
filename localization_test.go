package main

import (
	"encoding/json"
	"strings"
	"testing"
)

type localeCatalog struct {
	Meta struct {
		Code     string `json:"code"`
		Name     string `json:"name"`
		Fallback string `json:"fallback"`
	} `json:"meta"`
	Messages map[string]string `json:"messages"`
}

func readLocaleCatalog(t *testing.T, name string) localeCatalog {
	t.Helper()
	data, err := localeFS.ReadFile("locales/" + name)
	if err != nil {
		t.Fatalf("read locale %s: %v", name, err)
	}
	var catalog localeCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		t.Fatalf("parse locale %s: %v", name, err)
	}
	return catalog
}

func TestLocalizationCatalogs(t *testing.T) {
	manifest, err := loadLocaleManifest()
	if err != nil {
		t.Fatalf("load locale manifest: %v", err)
	}
	if manifest.Fallback != "en" || len(manifest.Languages) != 2 || manifest.Languages[0].Code != "en" || manifest.Languages[1].Code != "uk" {
		t.Fatalf("unexpected locale manifest: %+v", manifest)
	}
	en := readLocaleCatalog(t, "en.json")
	uk := readLocaleCatalog(t, "uk.json")
	if en.Meta.Code != "en" || uk.Meta.Code != "uk" {
		t.Fatalf("unexpected locale codes: en=%q uk=%q", en.Meta.Code, uk.Meta.Code)
	}
	if uk.Meta.Fallback != "en" {
		t.Fatalf("Ukrainian fallback = %q, want en", uk.Meta.Fallback)
	}
	if uk.Messages["Language"] != "Мова" || uk.Messages["Open project"] != "Відкрити проєкт" || uk.Messages["Choose language"] != "Обрати мову" {
		t.Fatal("core Ukrainian UI translations are missing")
	}
	if len(uk.Messages)*100 < len(en.Messages)*80 {
		t.Fatalf("Ukrainian coverage too low: %d/%d", len(uk.Messages), len(en.Messages))
	}
	if _, err := localeFS.ReadFile("locales/ru.json"); err == nil {
		t.Fatal("Russian localization must not be bundled")
	}
}

func TestDesktopLocalizationBootstrap(t *testing.T) {
	if !strings.Contains(appHTML, "data-i18n-language-host") {
		t.Fatal("desktop language toggle host is missing")
	}
	if !strings.Contains(appHTML, "./i18n.js") {
		t.Fatal("desktop localization runtime is not loaded")
	}
	if !strings.Contains(appHTML, "__APP_LANGUAGE__") {
		t.Fatal("desktop saved-language placeholder is missing")
	}
	if !strings.Contains(appHTML, "__APP_FLAG_MODE__") {
		t.Fatal("desktop flag-mode placeholder is missing")
	}
	if len(appI18nJS) == 0 {
		t.Fatal("desktop localization runtime is empty")
	}
	if !strings.Contains(string(appI18nJS), "X-GBA-Token") {
		t.Fatal("desktop language persistence does not use the API session token")
	}
	if !strings.Contains(string(appI18nJS), "language-menu-button") || !strings.Contains(string(appI18nJS), "language-menu-option") || !strings.Contains(string(appI18nJS), "locales/index.json") {
		t.Fatal("desktop localization must use the manifest-driven language dropdown")
	}
	if strings.Contains(string(appI18nJS), "setLanguage(language ===") {
		t.Fatal("desktop language button must open a menu instead of directly toggling languages")
	}
	if normalizeAppLanguage("en") != "en" || normalizeAppLanguage("en-US") != "en" || normalizeAppLanguage("uk") != "uk" || normalizeAppLanguage("ru") != "" {
		t.Fatal("desktop language allowlist must come from the EN/UK locale manifest")
	}
	if !localeAssetAllowed("index.json") || !localeAssetAllowed("en.json") || !localeAssetAllowed("uk.json") || !localeAssetAllowed("flag-gb.svg") || !localeAssetAllowed("flag-ua.svg") || localeAssetAllowed("ru.json") {
		t.Fatal("desktop locale route must follow the locale manifest")
	}
}
