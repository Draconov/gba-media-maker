package main

import (
	"encoding/binary"
	"testing"
)

func TestDefaultTitleCardSettingsUseSourceNameAndPartToken(t *testing.T) {
	settings := defaultTitleCardSettings("My long holiday video.mp4")
	if settings.Title != "MY LONG HOLIDAY VIDEO" {
		t.Fatalf("title=%q", settings.Title)
	}
	if settings.Subtitle != "Part {part}" {
		t.Fatalf("subtitle=%q", settings.Subtitle)
	}
	if settings.BackgroundMode != "part-first-frame" || settings.Darkness != 50 || settings.TextSize != "large" {
		t.Fatalf("unexpected defaults: %+v", settings)
	}
}

func TestResolveTitleCardSettingsSupportsSharedAndPerPartValues(t *testing.T) {
	project := &TitleCardProjectSettings{
		Enabled:   true,
		UseShared: false,
		Shared:    defaultTitleCardSettings("movie.mp4"),
		Parts: []TitleCardPartSettings{{
			Part: 2,
			Settings: TitleCardSettings{
				Title: "Second disc", Subtitle: "Part {part}", BackgroundMode: "solid",
				SolidColor: "#123456", TextColor: "#FFFFFF", OutlineColor: "#000000",
				DrawOutline: true, Alignment: "center", StartMode: "timer", DurationSeconds: 2,
				AllowSkip: true, Fade: true,
			},
		}},
	}
	settings, ok := resolveTitleCardSettings(project, "movie.mp4", 2)
	if !ok {
		t.Fatal("title card was not resolved")
	}
	if settings.Title != "SECOND DISC" || settings.Subtitle != "PART 2" {
		t.Fatalf("resolved text=%q / %q", settings.Title, settings.Subtitle)
	}
	if settings.BackgroundMode != "solid" || settings.StartMode != "timer" {
		t.Fatalf("resolved settings=%+v", settings)
	}
}

func TestBuildTitleCardAssetCreatesNativeMode3Payload(t *testing.T) {
	settings := defaultTitleCardSettings("movie.mp4")
	settings.StartMode = "timer"
	settings.DurationSeconds = 2
	background := solidTitleCardBackground("#204060")
	asset, err := buildTitleCardAsset(background, normalizeTitleCardSettings(settings, "movie.mp4", 3))
	if err != nil {
		t.Fatal(err)
	}
	if len(asset) != titleCardHeaderSize+titleCardPixelBytes {
		t.Fatalf("asset size=%d", len(asset))
	}
	if binary.LittleEndian.Uint32(asset[0:4]) != titleCardMagic {
		t.Fatalf("magic=%08x", binary.LittleEndian.Uint32(asset[0:4]))
	}
	if binary.LittleEndian.Uint16(asset[4:6]) != 1 {
		t.Fatalf("version=%d", binary.LittleEndian.Uint16(asset[4:6]))
	}
	flags := binary.LittleEndian.Uint16(asset[6:8])
	if flags&titleCardFlagWaitForA != 0 || flags&titleCardFlagAllowSkip == 0 || flags&titleCardFlagFade == 0 {
		t.Fatalf("flags=%04x", flags)
	}
	if binary.LittleEndian.Uint32(asset[8:12]) != titleCardPixelBytes {
		t.Fatalf("pixel bytes=%d", binary.LittleEndian.Uint32(asset[8:12]))
	}
	if got := binary.LittleEndian.Uint32(asset[12:16]); got < 119 || got > 120 {
		t.Fatalf("duration vblanks=%d", got)
	}
	allZero := true
	for _, value := range asset[titleCardHeaderSize:] {
		if value != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Fatal("rendered title-card pixels are empty")
	}
}

func TestTitleCardTextSizeChangesRenderedTypography(t *testing.T) {
	background := solidTitleCardBackground("#000000")
	settings := defaultTitleCardSettings("A VERY LONG VIDEO TITLE.mp4")
	settings.Subtitle = "PART 1"

	settings.TextSize = "large"
	large, err := renderTitleCardPixels(background, normalizeTitleCardSettings(settings, "movie.mp4", 1))
	if err != nil {
		t.Fatal(err)
	}
	settings.TextSize = "small"
	small, err := renderTitleCardPixels(background, normalizeTitleCardSettings(settings, "movie.mp4", 1))
	if err != nil {
		t.Fatal(err)
	}
	if string(large) == string(small) {
		t.Fatal("large and small title-card text rendered identically")
	}
}
