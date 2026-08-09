package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestPinnedFFmpegArchiveIsDiscoveredFromChecksums(t *testing.T) {
	workflow, err := os.ReadFile(".github/workflows/release.yml")
	if err != nil {
		t.Fatal(err)
	}
	fetchScript, err := os.ReadFile("scripts/fetch-ffmpeg.ps1")
	if err != nil {
		t.Fatal(err)
	}

	for name, source := range map[string]string{
		"release workflow":        string(workflow),
		"PowerShell fetch helper": string(fetchScript),
	} {
		if strings.Contains(source, `ffmpeg-master-latest-win64-lgpl.zip`) {
			t.Fatalf("%s hardcodes the floating latest-release filename for a dated BtbN release", name)
		}
		if strings.Contains(source, `ffmpeg-master-`) {
			t.Fatalf("%s incorrectly assumes dated BtbN master builds contain the word master in the asset filename", name)
		}
		if !strings.Contains(source, `checksums.sha256`) || !strings.Contains(source, `win64-lgpl`) {
			t.Fatalf("%s does not discover the pinned win64 LGPL archive from checksums.sha256", name)
		}
	}
}

func TestPinnedBtbNChecksumNamingFixtureSelectsNonSharedMasterBuild(t *testing.T) {
	// This mirrors the asset names printed by BtbN's dated autobuild checksum
	// manifest. The master snapshot is named ffmpeg-N-..., not ffmpeg-master-....
	fixture := strings.Join([]string{
		"f7b3b99cb0d4cc77baf13ff6c451ce564c904686eec228b7812a0aaff501c14d  ffmpeg-N-125990-g5c395992f9-win64-lgpl-shared.zip",
		"c7ae39256aacd89a1204d62293f0da6c6623497684117844aaec758c59729673  ffmpeg-N-125990-g5c395992f9-win64-lgpl.zip",
		"0dc7006a5b4207e171873c7a66ec377f26402ba8d7b77d06de317366272d600a  ffmpeg-n7.1.5-12-g1fbdca85aa-win64-lgpl-7.1.zip",
		"9e443926cf398900992999c548bd6af1c21dd8ddac30bf8db23f237ec535c0ea  ffmpeg-n7.1.5-12-g1fbdca85aa-win64-lgpl-shared-7.1.zip",
	}, "\n")

	re := regexp.MustCompile(`(?m)^[0-9a-fA-F]{64}\s+\*?(ffmpeg-\S+-win64-lgpl\.zip)$`)
	match := re.FindStringSubmatch(fixture)
	if len(match) != 2 {
		t.Fatal("did not discover the pinned non-shared win64 LGPL archive")
	}
	if got, want := match[1], "ffmpeg-N-125990-g5c395992f9-win64-lgpl.zip"; got != want {
		t.Fatalf("selected archive %q, want %q", got, want)
	}
}
