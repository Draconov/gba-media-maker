package main

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

var localePlaceholderPattern = regexp.MustCompile(`\{[a-zA-Z0-9_]+\}`)

func placeholderSet(value string) map[string]bool {
	out := map[string]bool{}
	for _, match := range localePlaceholderPattern.FindAllString(value, -1) {
		out[match] = true
	}
	return out
}

func samePlaceholderSet(a, b string) bool {
	left := placeholderSet(a)
	right := placeholderSet(b)
	if len(left) != len(right) {
		return false
	}
	for placeholder := range left {
		if !right[placeholder] {
			return false
		}
	}
	return true
}

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
	if manifest.Fallback != "en" || len(manifest.Languages) != 5 || manifest.Languages[0].Code != "en" || manifest.Languages[1].Code != "uk" || manifest.Languages[2].Code != "fr" || manifest.Languages[3].Code != "es" || manifest.Languages[4].Code != "de" {
		t.Fatalf("unexpected locale manifest: %+v", manifest)
	}
	en := readLocaleCatalog(t, "en.json")
	uk := readLocaleCatalog(t, "uk.json")
	fr := readLocaleCatalog(t, "fr.json")
	es := readLocaleCatalog(t, "es.json")
	de := readLocaleCatalog(t, "de.json")
	if en.Meta.Code != "en" || uk.Meta.Code != "uk" || fr.Meta.Code != "fr" || es.Meta.Code != "es" || de.Meta.Code != "de" {
		t.Fatalf("unexpected locale codes: en=%q uk=%q fr=%q es=%q de=%q", en.Meta.Code, uk.Meta.Code, fr.Meta.Code, es.Meta.Code, de.Meta.Code)
	}
	for name, catalog := range map[string]localeCatalog{"uk": uk, "fr": fr, "es": es, "de": de} {
		if catalog.Meta.Fallback != "en" {
			t.Fatalf("%s fallback = %q, want en", name, catalog.Meta.Fallback)
		}
	}
	if uk.Messages["Language"] != "Мова" || uk.Messages["Open project"] != "Відкрити проєкт" || uk.Messages["Choose language"] != "Обрати мову" {
		t.Fatal("core Ukrainian UI translations are missing")
	}
	for name, catalog := range map[string]localeCatalog{"uk": uk, "fr": fr, "es": es, "de": de} {
		if len(catalog.Messages)*100 < len(en.Messages)*80 {
			t.Fatalf("%s coverage too low: %d/%d", name, len(catalog.Messages), len(en.Messages))
		}
	}
	if fr.Messages["Language"] != "Langue" || es.Messages["Language"] != "Idioma" || de.Messages["Language"] != "Sprache" {
		t.Fatal("French, Spanish, or German core UI translations are missing")
	}
	for name, catalog := range map[string]localeCatalog{"fr": fr, "es": es, "de": de} {
		if len(catalog.Messages) != len(en.Messages) {
			t.Fatalf("%s catalog is incomplete: %d/%d messages", name, len(catalog.Messages), len(en.Messages))
		}
		for key := range en.Messages {
			translated, ok := catalog.Messages[key]
			if !ok {
				t.Fatalf("%s catalog is missing %q", name, key)
			}
			if !samePlaceholderSet(key, translated) {
				t.Fatalf("%s placeholders differ for %q: %q", name, key, translated)
			}
		}
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
	if normalizeAppLanguage("en") != "en" || normalizeAppLanguage("en-US") != "en" || normalizeAppLanguage("uk") != "uk" || normalizeAppLanguage("fr-FR") != "fr" || normalizeAppLanguage("es-ES") != "es" || normalizeAppLanguage("de-DE") != "de" || normalizeAppLanguage("ru") != "" {
		t.Fatal("desktop language allowlist must come from the locale manifest")
	}
	if !localeAssetAllowed("index.json") || !localeAssetAllowed("en.json") || !localeAssetAllowed("uk.json") || !localeAssetAllowed("fr.json") || !localeAssetAllowed("es.json") || !localeAssetAllowed("de.json") || !localeAssetAllowed("flag-gb.svg") || !localeAssetAllowed("flag-ua.svg") || !localeAssetAllowed("flag-fr.svg") || !localeAssetAllowed("flag-es.svg") || !localeAssetAllowed("flag-de.svg") || localeAssetAllowed("ru.json") {
		t.Fatal("desktop locale route must follow the locale manifest")
	}
}
