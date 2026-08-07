package main

import "testing"

func TestSmartCandidateGenerationRespectsTargetAndAudioChoice(t *testing.T) {
	candidates := buildSmartCandidates(12*60, true, audioCodecAuto, 32<<20, "balanced", .25, .4, .35, .16)
	if len(candidates) < 4 {
		t.Fatalf("got %d candidates", len(candidates))
	}
	if candidates[0].EstimatedBytes <= 0 || candidates[0].VisualQuality <= 0 {
		t.Fatalf("invalid recommendation: %#v", candidates[0])
	}
	seenPCM, seenADPCM := false, false
	for _, candidate := range candidates {
		seenPCM = seenPCM || candidate.AudioCodec == audioCodecPCM
		seenADPCM = seenADPCM || candidate.AudioCodec == audioCodecADPCM
	}
	if !seenPCM || !seenADPCM {
		t.Fatalf("auto candidates did not expose both audio choices: %#v", candidates)
	}
}

func TestRepresentativeSamplesAreDiverseAndOrdered(t *testing.T) {
	metrics := make([]smartFrameMetric, 40)
	for i := range metrics {
		metrics[i] = smartFrameMetric{index: i, time: float64(i), motion: float64(i%10) / 10, detail: float64((i*3)%10) / 10, brightness: float64((i*7)%10) / 10, colour: float64((i*5)%10) / 10, scene: float64((i*9)%10) / 10}
	}
	samples := selectSmartSamples(metrics)
	if len(samples) < 4 || len(samples) > smartSampleCount {
		t.Fatalf("unexpected sample count %d", len(samples))
	}
	for i := 1; i < len(samples); i++ {
		if samples[i].Time < samples[i-1].Time {
			t.Fatal("samples are not ordered")
		}
	}
}
