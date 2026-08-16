package main

import "testing"

func TestAutomaticAudioArtworkPresetIsStableAndVaried(t *testing.T) {
	seeds := []string{"alpha.mp3", "beta.mp3", "gamma.mp3", "delta.mp3", "epsilon.mp3", "zeta.mp3"}
	seen := map[string]bool{}
	for _, seed := range seeds {
		first := automaticAudioArtworkPreset(seed)
		second := automaticAudioArtworkPreset(seed)
		if first != second {
			t.Fatalf("automatic preset is not stable for %q: %q != %q", seed, first, second)
		}
		if !audioArtworkPresetPattern.MatchString(first) {
			t.Fatalf("automatic preset is outside built-in selection: %q", first)
		}
		seen[first] = true
	}
	if len(seen) < 2 {
		t.Fatalf("automatic artwork assignment did not vary across test tracks: %v", seen)
	}
}
