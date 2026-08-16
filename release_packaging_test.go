package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestPinnedFFmpegArchiveIsDiscoveredFromChecksums(t *testing.T) {
	fetchShell, err := os.ReadFile("scripts/fetch-ffmpeg-btbn.sh")
	if err != nil {
		t.Fatal(err)
	}
	fetchPowerShell, err := os.ReadFile("scripts/fetch-ffmpeg.ps1")
	if err != nil {
		t.Fatal(err)
	}
	pins, err := os.ReadFile("scripts/ffmpeg-pins.env")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(pins), "BTBN_FFMPEG_TAG=autobuild-") {
		t.Fatal("central FFmpeg pin file does not define a dated BtbN release")
	}
	for name, source := range map[string]string{
		"cross-platform fetch helper": string(fetchShell),
		"PowerShell fetch helper":     string(fetchPowerShell),
	} {
		if strings.Contains(source, `ffmpeg-master-latest-win64-lgpl.zip`) {
			t.Fatalf("%s hardcodes a floating latest-release filename", name)
		}
		if !strings.Contains(source, `checksums.sha256`) {
			t.Fatalf("%s does not use the pinned checksum manifest", name)
		}
	}
	for _, suffix := range []string{"win64-lgpl", "linux64-lgpl"} {
		if !strings.Contains(string(fetchShell), suffix) {
			t.Fatalf("cross-platform fetch helper does not support %s", suffix)
		}
	}
}

func TestPinnedBtbNChecksumNamingFixtureSelectsNonSharedBuilds(t *testing.T) {
	fixture := strings.Join([]string{
		"f7b3b99cb0d4cc77baf13ff6c451ce564c904686eec228b7812a0aaff501c14d  ffmpeg-N-125990-g5c395992f9-win64-lgpl-shared.zip",
		"c7ae39256aacd89a1204d62293f0da6c6623497684117844aaec758c59729673  ffmpeg-N-125990-g5c395992f9-win64-lgpl.zip",
		"f7b3b99cb0d4cc77baf13ff6c451ce564c904686eec228b7812a0aaff501c14d  ffmpeg-N-125990-g5c395992f9-linux64-lgpl-shared.tar.xz",
		"c7ae39256aacd89a1204d62293f0da6c6623497684117844aaec758c59729673  ffmpeg-N-125990-g5c395992f9-linux64-lgpl.tar.xz",
	}, "\n")

	for _, suffix := range []string{`win64-lgpl\.zip`, `linux64-lgpl\.tar\.xz`} {
		re := regexp.MustCompile(`(?m)^[0-9a-fA-F]{64}\s+\*?(ffmpeg-\S+-` + suffix + `)$`)
		match := re.FindStringSubmatch(fixture)
		if len(match) != 2 {
			t.Fatalf("did not discover pinned non-shared archive matching %s", suffix)
		}
		if strings.Contains(match[1], "shared") {
			t.Fatalf("selected shared archive %q", match[1])
		}
	}
}

func TestReleaseWorkflowPublishesDesktopPackages(t *testing.T) {
	workflow, err := os.ReadFile(".github/workflows/release.yml")
	if err != nil {
		t.Fatal(err)
	}
	source := string(workflow)
	for _, want := range []string{
		"windows:", "linux:", "macos-slices:", "macos:",
		"macos-15-intel", "macos-15",
		"build-linux-slice.sh amd64", "package-linux.sh", "build-macos-slice.sh", "package-macos.sh", "build-ffmpeg-macos.sh",
		"_Linux_x86_64.tar.gz", "_macOS.zip", "Expected three release archives plus three checksums",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("release workflow is missing %q", want)
		}
	}
	for _, unwanted := range []string{"linux-slices:", "ubuntu-24.04-arm", "linux-slice-arm64", "_Linux_ARM64.tar.gz", "_macOS_Intel.zip", "_macOS_AppleSilicon.zip"} {
		if strings.Contains(source, unwanted) {
			t.Fatalf("release workflow still contains unsupported/per-architecture release path %q", unwanted)
		}
	}
}

func TestLinuxX8664AndMacUniversalPackageScripts(t *testing.T) {
	linux, err := os.ReadFile("scripts/package-linux.sh")
	if err != nil {
		t.Fatal(err)
	}
	macos, err := os.ReadFile("scripts/package-macos.sh")
	if err != nil {
		t.Fatal(err)
	}
	linuxSource := string(linux)
	for _, want := range []string{"Linux_x86_64", "verify_x64", "gba-media-maker", "ffmpeg", "install-user.sh"} {
		if !strings.Contains(linuxSource, want) {
			t.Fatalf("Linux x86_64 packager is missing %q", want)
		}
	}
	for _, unwanted := range []string{"bin/arm64", "ARM64_SLICE", "aarch64|arm64"} {
		if strings.Contains(linuxSource, unwanted) {
			t.Fatalf("Linux x86_64 packager still contains universal/ARM64 path %q", unwanted)
		}
	}
	macSource := string(macos)
	for _, want := range []string{"lipo -create", "x86_64", "arm64", "GBA_Media_Maker_v${VERSION}_macOS.zip"} {
		if !strings.Contains(macSource, want) {
			t.Fatalf("macOS Universal 2 packager is missing %q", want)
		}
	}
}

func TestWindowsReleaseUsesPlatformName(t *testing.T) {
	workflow, err := os.ReadFile(".github/workflows/release.yml")
	if err != nil {
		t.Fatal(err)
	}
	packager, err := os.ReadFile("scripts/package-release.ps1")
	if err != nil {
		t.Fatal(err)
	}
	for name, source := range map[string]string{
		"release workflow":    string(workflow),
		"PowerShell packager": string(packager),
	} {
		if strings.Contains(source, "_Portable") {
			t.Fatalf("%s still uses the ambiguous Portable release name", name)
		}
		if !strings.Contains(source, "_Windows_x64") {
			t.Fatalf("%s does not use the Windows_x64 release name", name)
		}
	}
}

func TestReleaseWorkflowsDoNotDependOnShellScriptExecutableBits(t *testing.T) {
	for _, path := range []string{".github/workflows/release.yml", ".github/workflows/ci.yml"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		source := string(data)
		// GitHub preserves the executable bit only when it is committed as file
		// mode 100755. Archive/web uploads can turn scripts into 100644, so CI
		// must invoke repository shell scripts through bash explicitly.
		directScript := regexp.MustCompile(`(?m)^\s*(?:run:\s*)?\./scripts/[A-Za-z0-9_.-]+\.sh(?:\s|$)`)
		if match := directScript.FindString(source); match != "" {
			t.Fatalf("%s directly executes a repository shell script and can fail with permission denied: %q", path, strings.TrimSpace(match))
		}
	}
}

func TestWindowsReleaseIconHelperRunsForHostOS(t *testing.T) {
	workflow, err := os.ReadFile(".github/workflows/release.yml")
	if err != nil {
		t.Fatal(err)
	}
	source := string(workflow)
	buildStart := strings.Index(source, "      - name: Build Windows executable")
	embedStart := strings.Index(source, "      - name: Embed application icon")
	fetchStart := strings.Index(source, "      - name: Fetch pinned LGPL FFmpeg")
	if buildStart < 0 || embedStart <= buildStart || fetchStart <= embedStart {
		t.Fatal("Windows build/icon/fetch steps are not separated correctly")
	}
	buildBlock := source[buildStart:embedStart]
	embedBlock := source[embedStart:fetchStart]
	if !strings.Contains(buildBlock, "GOOS: windows") || strings.Contains(buildBlock, "tools/embedicon") {
		t.Fatal("Windows cross-build step must build only the Windows application")
	}
	if !strings.Contains(embedBlock, "go run ./tools/embedicon") || strings.Contains(embedBlock, "GOOS: windows") {
		t.Fatal("icon helper must run in a separate host-native step without GOOS=windows")
	}
}

func TestMacOSFFmpegPathsMatchSliceAndFinalBundleLayouts(t *testing.T) {
	builder, err := os.ReadFile("scripts/build-ffmpeg-macos.sh")
	if err != nil {
		t.Fatal(err)
	}
	packager, err := os.ReadFile("scripts/package-macos.sh")
	if err != nil {
		t.Fatal(err)
	}
	builderSource := string(builder)
	packagerSource := string(packager)
	if !strings.Contains(builderSource, `@executable_path/Frameworks/$base`) {
		t.Fatal("macOS FFmpeg slice must resolve bundled frameworks beside the slice executable")
	}
	if !strings.Contains(builderSource, `@loader_path/$(basename "$dylib")`) {
		t.Fatal("bundled macOS dylibs must use loader-relative install names")
	}
	if !strings.Contains(packagerSource, `@executable_path/../Frameworks/$base`) {
		t.Fatal("Universal macOS packager must rewrite FFmpeg frameworks for Contents/MacOS")
	}
	if !strings.Contains(packagerSource, `@executable_path/Frameworks/`) {
		t.Fatal("Universal macOS package verification must reject unrevised slice paths")
	}
}
