package main

import (
	"os"
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
		if !strings.Contains(source, `ffmpeg-master-`) || !strings.Contains(source, `win64-lgpl`) || !strings.Contains(source, `checksums.sha256`) {
			t.Fatalf("%s does not discover the pinned win64 LGPL archive from checksums.sha256", name)
		}
	}
}
