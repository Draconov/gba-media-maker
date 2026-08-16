package main

import "testing"

func TestCLIInvocationDetection(t *testing.T) {
	for _, args := range [][]string{{"-input", "movie.mp4"}, {"--input=movie.mp4"}, {"--cli"}, {"--help"}} {
		if !wantsCLI(args) {
			t.Fatalf("wantsCLI(%q) = false", args)
		}
	}
	if wantsCLI(nil) || wantsCLI([]string{"-psn_0_12345"}) {
		t.Fatal("desktop launch arguments must not accidentally force CLI mode")
	}
}

func TestDialogFilterParsing(t *testing.T) {
	groups := parseDialogFilter(projectDialogFilter)
	if len(groups) != 2 {
		t.Fatalf("project dialog groups = %d, want 2", len(groups))
	}
	if groups[0].Name != "GBA Media Maker project" || len(groups[0].Patterns) != 2 {
		t.Fatalf("unexpected project filter: %+v", groups[0])
	}
	if groups[0].Patterns[0] != "*.gbamedia" || groups[0].Patterns[1] != "*.gbavideo" {
		t.Fatalf("unexpected project patterns: %#v", groups[0].Patterns)
	}
}

func TestDialogOutputParsing(t *testing.T) {
	got := joinDialogOutput("/tmp/a.mp4\r\n/tmp/b.mp3\n")
	if len(got) != 2 || got[0] != "/tmp/a.mp4" || got[1] != "/tmp/b.mp3" {
		t.Fatalf("unexpected dialog output: %#v", got)
	}
}

func TestDialogExtension(t *testing.T) {
	if got := ensureDialogExtension("/tmp/My Project", "gbamedia"); got != "/tmp/My Project.gbamedia" {
		t.Fatalf("ensureDialogExtension() = %q", got)
	}
	if got := ensureDialogExtension("/tmp/My Project.gbamedia", "gbamedia"); got != "/tmp/My Project.gbamedia" {
		t.Fatalf("existing extension changed: %q", got)
	}
}
