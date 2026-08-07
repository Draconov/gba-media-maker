package main

import (
	"math"
	"testing"
)

func TestIMAADPCMRoundTrip(t *testing.T) {
	pcm := make([]byte, audioRate*2+137)
	for i := range pcm {
		v := math.Sin(float64(i)*2*math.Pi*440/audioRate)*78 + math.Sin(float64(i)*2*math.Pi*83/audioRate)*22
		pcm[i] = byte(int8(v))
	}
	encoded, info, err := encodeIMAADPCM(pcm, defaultADPCMBlockSamples)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) >= len(pcm)*3/5 {
		t.Fatalf("ADPCM did not reduce storage enough: %d vs %d", len(encoded), len(pcm))
	}
	decoded, decodedInfo, err := decodeIMAADPCM(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decodedInfo != info || len(decoded) != len(pcm) {
		t.Fatalf("metadata mismatch: %#v %#v", info, decodedInfo)
	}
	var mse float64
	for i := range pcm {
		d := float64(int(int8(pcm[i])) - int(int8(decoded[i])))
		mse += d * d
	}
	mse /= float64(len(pcm))
	if mse > 350 {
		t.Fatalf("unexpectedly poor ADPCM quality, MSE %.2f", mse)
	}
}

func TestResolveAudioCodecKeepsLegacyPCM(t *testing.T) {
	if got := resolveAudioCodec(audioCodecAuto, false, 20<<20, 32<<20); got != audioCodecPCM {
		t.Fatalf("legacy mode chose %q", got)
	}
	if got := resolveAudioCodec(audioCodecAuto, true, 20<<20, 32<<20); got != audioCodecADPCM {
		t.Fatalf("extreme mode chose %q", got)
	}
}
