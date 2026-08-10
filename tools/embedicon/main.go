package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"os"
)

const (
	resourceTypeIcon      = 3
	resourceTypeGroupIcon = 14
	languageEnglishUS     = 1033
	resourceDataRead      = 0x40000040 // initialized data + readable
)

type iconEntry struct {
	width, height, colors, reserved byte
	planes, bits                    uint16
	data                            []byte
}

func align(v, a uint32) uint32 {
	if a == 0 {
		return v
	}
	return (v + a - 1) &^ (a - 1)
}

func parseICO(data []byte) ([]iconEntry, error) {
	if len(data) < 6 || binary.LittleEndian.Uint16(data[0:2]) != 0 || binary.LittleEndian.Uint16(data[2:4]) != 1 {
		return nil, errors.New("not a Windows ICO file")
	}
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	if count < 1 || len(data) < 6+count*16 {
		return nil, errors.New("invalid ICO directory")
	}
	entries := make([]iconEntry, 0, count)
	for i := 0; i < count; i++ {
		o := 6 + i*16
		sz := int(binary.LittleEndian.Uint32(data[o+8 : o+12]))
		off := int(binary.LittleEndian.Uint32(data[o+12 : o+16]))
		if sz <= 0 || off < 0 || off+sz > len(data) {
			return nil, fmt.Errorf("invalid ICO image %d", i)
		}
		entries = append(entries, iconEntry{
			width: data[o], height: data[o+1], colors: data[o+2], reserved: data[o+3],
			planes: binary.LittleEndian.Uint16(data[o+4 : o+6]),
			bits:   binary.LittleEndian.Uint16(data[o+6 : o+8]),
			data:   append([]byte(nil), data[off:off+sz]...),
		})
	}
	return entries, nil
}

func putDir(b []byte, off int, idCount uint16) {
	binary.LittleEndian.PutUint16(b[off+14:off+16], idCount)
}

func putEntry(b []byte, off int, id uint32, target uint32, directory bool) {
	binary.LittleEndian.PutUint32(b[off:off+4], id)
	if directory {
		target |= 0x80000000
	}
	binary.LittleEndian.PutUint32(b[off+4:off+8], target)
}

func buildResource(entries []iconEntry, baseRVA uint32) []byte {
	n := len(entries)
	rootOff := 0
	rootEntriesOff := 16
	iconDirOff := rootEntriesOff + 16
	iconLangOff := iconDirOff + 16 + n*8
	groupDirOff := iconLangOff + n*24
	groupLangOff := groupDirOff + 24
	dataEntriesOff := groupLangOff + 24
	blobOff := int(align(uint32(dataEntriesOff+(n+1)*16), 4))
	iconBlobOffsets := make([]int, n)
	cursor := blobOff
	for i, e := range entries {
		iconBlobOffsets[i] = cursor
		cursor = int(align(uint32(cursor+len(e.data)), 4))
	}
	groupBlobOff := cursor
	groupSize := 6 + n*14
	cursor = int(align(uint32(cursor+groupSize), 4))
	out := make([]byte, cursor)

	putDir(out, rootOff, 2)
	putEntry(out, rootEntriesOff, resourceTypeIcon, uint32(iconDirOff), true)
	putEntry(out, rootEntriesOff+8, resourceTypeGroupIcon, uint32(groupDirOff), true)
	putDir(out, iconDirOff, uint16(n))
	for i := range entries {
		langDir := iconLangOff + i*24
		putEntry(out, iconDirOff+16+i*8, uint32(i+1), uint32(langDir), true)
		putDir(out, langDir, 1)
		dataEntry := dataEntriesOff + i*16
		putEntry(out, langDir+16, languageEnglishUS, uint32(dataEntry), false)
		binary.LittleEndian.PutUint32(out[dataEntry:dataEntry+4], baseRVA+uint32(iconBlobOffsets[i]))
		binary.LittleEndian.PutUint32(out[dataEntry+4:dataEntry+8], uint32(len(entries[i].data)))
		copy(out[iconBlobOffsets[i]:], entries[i].data)
	}
	putDir(out, groupDirOff, 1)
	putEntry(out, groupDirOff+16, 1, uint32(groupLangOff), true)
	putDir(out, groupLangOff, 1)
	groupDataEntry := dataEntriesOff + n*16
	putEntry(out, groupLangOff+16, languageEnglishUS, uint32(groupDataEntry), false)
	binary.LittleEndian.PutUint32(out[groupDataEntry:groupDataEntry+4], baseRVA+uint32(groupBlobOff))
	binary.LittleEndian.PutUint32(out[groupDataEntry+4:groupDataEntry+8], uint32(groupSize))
	g := out[groupBlobOff : groupBlobOff+groupSize]
	binary.LittleEndian.PutUint16(g[2:4], 1)
	binary.LittleEndian.PutUint16(g[4:6], uint16(n))
	for i, e := range entries {
		o := 6 + i*14
		g[o], g[o+1], g[o+2], g[o+3] = e.width, e.height, e.colors, e.reserved
		binary.LittleEndian.PutUint16(g[o+4:o+6], e.planes)
		binary.LittleEndian.PutUint16(g[o+6:o+8], e.bits)
		binary.LittleEndian.PutUint32(g[o+8:o+12], uint32(len(e.data)))
		binary.LittleEndian.PutUint16(g[o+12:o+14], uint16(i+1))
	}
	return out
}

func embed(exePath, icoPath string) error {
	exe, err := os.ReadFile(exePath)
	if err != nil {
		return err
	}
	ico, err := os.ReadFile(icoPath)
	if err != nil {
		return err
	}
	icons, err := parseICO(ico)
	if err != nil {
		return err
	}
	if len(exe) < 0x40 {
		return errors.New("executable is too small")
	}
	pe := int(binary.LittleEndian.Uint32(exe[0x3c:0x40]))
	if pe < 0 || pe+24 > len(exe) || string(exe[pe:pe+4]) != "PE\x00\x00" {
		return errors.New("not a PE executable")
	}
	coff := pe + 4
	sections := int(binary.LittleEndian.Uint16(exe[coff+2 : coff+4]))
	optSize := int(binary.LittleEndian.Uint16(exe[coff+16 : coff+18]))
	opt := coff + 20
	if opt+optSize > len(exe) || optSize < 128 || binary.LittleEndian.Uint16(exe[opt:opt+2]) != 0x20b {
		return errors.New("expected a PE32+ executable")
	}
	sectionAlignment := binary.LittleEndian.Uint32(exe[opt+32 : opt+36])
	fileAlignment := binary.LittleEndian.Uint32(exe[opt+36 : opt+40])
	sizeOfImage := binary.LittleEndian.Uint32(exe[opt+56 : opt+60])
	sizeOfHeaders := binary.LittleEndian.Uint32(exe[opt+60 : opt+64])
	resourceDir := opt + 112 + 2*8
	if resourceDir+8 > opt+optSize {
		return errors.New("PE has no resource data-directory slot")
	}
	if binary.LittleEndian.Uint32(exe[resourceDir:resourceDir+4]) != 0 {
		return errors.New("executable already contains Windows resources")
	}
	sectionTable := opt + optSize
	newHeader := sectionTable + sections*40
	if newHeader+40 > int(sizeOfHeaders) || newHeader+40 > len(exe) {
		return errors.New("PE header has no room for another section")
	}
	newRVA := align(sizeOfImage, sectionAlignment)
	resource := buildResource(icons, newRVA)
	newRaw := align(uint32(len(exe)), fileAlignment)
	newRawSize := align(uint32(len(resource)), fileAlignment)
	need := int(newRaw + newRawSize)
	if need > len(exe) {
		exe = append(exe, make([]byte, need-len(exe))...)
	}
	copy(exe[newRaw:newRaw+uint32(len(resource))], resource)
	h := exe[newHeader : newHeader+40]
	copy(h[0:8], []byte(".rsrc\x00\x00\x00"))
	binary.LittleEndian.PutUint32(h[8:12], uint32(len(resource)))
	binary.LittleEndian.PutUint32(h[12:16], newRVA)
	binary.LittleEndian.PutUint32(h[16:20], newRawSize)
	binary.LittleEndian.PutUint32(h[20:24], newRaw)
	binary.LittleEndian.PutUint32(h[36:40], resourceDataRead)
	binary.LittleEndian.PutUint16(exe[coff+2:coff+4], uint16(sections+1))
	initialized := binary.LittleEndian.Uint32(exe[opt+8 : opt+12])
	binary.LittleEndian.PutUint32(exe[opt+8:opt+12], initialized+newRawSize)
	binary.LittleEndian.PutUint32(exe[opt+56:opt+60], align(newRVA+uint32(len(resource)), sectionAlignment))
	binary.LittleEndian.PutUint32(exe[opt+64:opt+68], 0)
	binary.LittleEndian.PutUint32(exe[resourceDir:resourceDir+4], newRVA)
	binary.LittleEndian.PutUint32(exe[resourceDir+4:resourceDir+8], uint32(len(resource)))
	tmp := exePath + ".icon.tmp"
	if err := os.WriteFile(tmp, exe, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmp, exePath); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func main() {
	exe := flag.String("exe", "GBA Media Maker.exe", "Windows executable to update")
	ico := flag.String("ico", "assets/app_icon.ico", "ICO file to embed")
	flag.Parse()
	if err := embed(*exe, *ico); err != nil {
		fmt.Fprintln(os.Stderr, "embedicon:", err)
		os.Exit(1)
	}
	fmt.Printf("Embedded %s into %s\n", *ico, *exe)
}
