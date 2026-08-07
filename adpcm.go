package main

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	audioCodecPCM            = "pcm"
	audioCodecADPCM          = "adpcm"
	audioCodecAuto           = "auto"
	adpcmMagic               = "IAD1"
	adpcmVersion             = 1
	defaultADPCMBlockSamples = 2048
	adpcmHeaderBytes         = 20
)

var imaIndexTable = [16]int{-1, -1, -1, -1, 2, 4, 6, 8, -1, -1, -1, -1, 2, 4, 6, 8}
var imaStepTable = [89]int{
	7, 8, 9, 10, 11, 12, 13, 14, 16, 17, 19, 21, 23, 25, 28, 31,
	34, 37, 41, 45, 50, 55, 60, 66, 73, 80, 88, 97, 107, 118, 130, 143,
	157, 173, 190, 209, 230, 253, 279, 307, 337, 371, 408, 449, 494, 544, 598, 658,
	724, 796, 876, 963, 1060, 1166, 1282, 1411, 1552, 1707, 1878, 2066, 2272, 2499, 2749, 3024,
	3327, 3660, 4026, 4428, 4871, 5358, 5894, 6484, 7132, 7845, 8630, 9493, 10442, 11487, 12635, 13899,
	15289, 16818, 18500, 20350, 22385, 24623, 27086, 29794, 32767,
}

type adpcmInfo struct {
	BlockSamples int
	SampleCount  int
	BlockBytes   int
	BlockCount   int
}

func clampInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func initialIMAIndex(pcm []byte, start, count int) int {
	if count < 2 || start+1 >= len(pcm) {
		return 0
	}
	maxDelta := 0
	limit := count
	if limit > 32 {
		limit = 32
	}
	prev := int(int8(pcm[start])) << 8
	for i := 1; i < limit && start+i < len(pcm); i++ {
		current := int(int8(pcm[start+i])) << 8
		delta := current - prev
		if delta < 0 {
			delta = -delta
		}
		if delta > maxDelta {
			maxDelta = delta
		}
		prev = current
	}
	index := 0
	for index < len(imaStepTable)-1 && imaStepTable[index] < maxDelta/2 {
		index++
	}
	return index
}

func encodeIMANibble(sample int, predictor *int, index *int) byte {
	step := imaStepTable[*index]
	diff := sample - *predictor
	code := 0
	if diff < 0 {
		code = 8
		diff = -diff
	}
	delta := step >> 3
	if diff >= step {
		code |= 4
		diff -= step
		delta += step
	}
	if diff >= step>>1 {
		code |= 2
		diff -= step >> 1
		delta += step >> 1
	}
	if diff >= step>>2 {
		code |= 1
		delta += step >> 2
	}
	if code&8 != 0 {
		*predictor -= delta
	} else {
		*predictor += delta
	}
	*predictor = clampInt(*predictor, -32768, 32767)
	*index = clampInt(*index+imaIndexTable[code], 0, 88)
	return byte(code)
}

func decodeIMANibble(code byte, predictor *int, index *int) int {
	step := imaStepTable[*index]
	delta := step >> 3
	if code&4 != 0 {
		delta += step
	}
	if code&2 != 0 {
		delta += step >> 1
	}
	if code&1 != 0 {
		delta += step >> 2
	}
	if code&8 != 0 {
		*predictor -= delta
	} else {
		*predictor += delta
	}
	*predictor = clampInt(*predictor, -32768, 32767)
	*index = clampInt(*index+imaIndexTable[code&15], 0, 88)
	return *predictor
}

func encodeIMAADPCM(pcm []byte, blockSamples int) ([]byte, adpcmInfo, error) {
	if blockSamples < 64 || blockSamples > 16384 {
		return nil, adpcmInfo{}, errors.New("ADPCM block size must be between 64 and 16384 samples")
	}
	blockBytes := 4 + (blockSamples-1+1)/2
	blockCount := 0
	if len(pcm) > 0 {
		blockCount = (len(pcm) + blockSamples - 1) / blockSamples
	}
	info := adpcmInfo{BlockSamples: blockSamples, SampleCount: len(pcm), BlockBytes: blockBytes, BlockCount: blockCount}
	out := make([]byte, adpcmHeaderBytes+blockCount*blockBytes)
	copy(out[:4], []byte(adpcmMagic))
	binary.LittleEndian.PutUint16(out[4:6], adpcmVersion)
	binary.LittleEndian.PutUint16(out[6:8], uint16(blockSamples))
	binary.LittleEndian.PutUint32(out[8:12], uint32(len(pcm)))
	binary.LittleEndian.PutUint32(out[12:16], uint32(blockBytes))
	binary.LittleEndian.PutUint32(out[16:20], uint32(blockCount))
	for block := 0; block < blockCount; block++ {
		start := block * blockSamples
		count := blockSamples
		if start+count > len(pcm) {
			count = len(pcm) - start
		}
		base := adpcmHeaderBytes + block*blockBytes
		predictor := int(int8(pcm[start])) << 8
		index := initialIMAIndex(pcm, start, count)
		binary.LittleEndian.PutUint16(out[base:base+2], uint16(int16(predictor)))
		out[base+2] = byte(index)
		out[base+3] = 0
		for i := 1; i < blockSamples; i++ {
			sample := predictor
			if i < count {
				sample = int(int8(pcm[start+i])) << 8
			}
			code := encodeIMANibble(sample, &predictor, &index)
			pos := base + 4 + (i-1)/2
			if (i-1)&1 == 0 {
				out[pos] = code & 15
			} else {
				out[pos] |= (code & 15) << 4
			}
		}
	}
	return out, info, nil
}

func parseADPCMInfo(data []byte) (adpcmInfo, error) {
	if len(data) < adpcmHeaderBytes || string(data[:4]) != adpcmMagic {
		return adpcmInfo{}, errors.New("invalid ADPCM stream")
	}
	if binary.LittleEndian.Uint16(data[4:6]) != adpcmVersion {
		return adpcmInfo{}, errors.New("unsupported ADPCM version")
	}
	info := adpcmInfo{
		BlockSamples: int(binary.LittleEndian.Uint16(data[6:8])),
		SampleCount:  int(binary.LittleEndian.Uint32(data[8:12])),
		BlockBytes:   int(binary.LittleEndian.Uint32(data[12:16])),
		BlockCount:   int(binary.LittleEndian.Uint32(data[16:20])),
	}
	if info.BlockSamples < 1 || info.BlockBytes < 4 || info.BlockCount < 0 || info.SampleCount < 0 {
		return adpcmInfo{}, errors.New("invalid ADPCM metadata")
	}
	if adpcmHeaderBytes+info.BlockBytes*info.BlockCount > len(data) {
		return adpcmInfo{}, errors.New("truncated ADPCM stream")
	}
	return info, nil
}

func decodeIMAADPCM(data []byte) ([]byte, adpcmInfo, error) {
	info, err := parseADPCMInfo(data)
	if err != nil {
		return nil, adpcmInfo{}, err
	}
	pcm := make([]byte, info.SampleCount)
	written := 0
	for block := 0; block < info.BlockCount && written < len(pcm); block++ {
		base := adpcmHeaderBytes + block*info.BlockBytes
		predictor := int(int16(binary.LittleEndian.Uint16(data[base : base+2])))
		index := clampInt(int(data[base+2]), 0, 88)
		pcm[written] = byte(int8(clampInt((predictor+128)>>8, -128, 127)))
		written++
		for i := 1; i < info.BlockSamples && written < len(pcm); i++ {
			pos := base + 4 + (i-1)/2
			code := data[pos] & 15
			if (i-1)&1 != 0 {
				code = (data[pos] >> 4) & 15
			}
			value := decodeIMANibble(code, &predictor, &index)
			pcm[written] = byte(int8(clampInt((value+128)>>8, -128, 127)))
			written++
		}
	}
	if written != len(pcm) {
		return nil, adpcmInfo{}, fmt.Errorf("decoded %d of %d samples", written, len(pcm))
	}
	return pcm, info, nil
}

func resolveAudioCodec(requested string, extreme bool, predictedPCMBytes, targetBytes int64) string {
	switch requested {
	case audioCodecADPCM:
		return audioCodecADPCM
	case audioCodecAuto:
		if extreme && predictedPCMBytes > 0 && targetBytes > 0 && predictedPCMBytes > targetBytes/3 {
			return audioCodecADPCM
		}
		return audioCodecPCM
	default:
		return audioCodecPCM
	}
}
