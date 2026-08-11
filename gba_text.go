package main

import (
	"strings"
	"unicode"
)

// GBA text uses one compact 3x5 bitmap font for Latin and the union of the
// Ukrainian and Russian Cyrillic alphabets. User/project strings remain UTF-8;
// runtime menu strings are converted to single-byte glyph IDs while building
// the ROM so the ARM7TDMI player never needs a Unicode decoder.

var gbaGlyphs = map[rune]uint16{
	'0': 0x7B6F, '1': 0x2C97, '2': 0x73E7, '3': 0x73CF, '4': 0x5BC9,
	'5': 0x79CF, '6': 0x79EF, '7': 0x7292, '8': 0x7BEF, '9': 0x7BCF,
	'A': 0x2BED, 'B': 0x6BAE, 'C': 0x7927, 'D': 0x6B6E, 'E': 0x79E7,
	'F': 0x79E4, 'G': 0x79AF, 'H': 0x5BED, 'I': 0x7497, 'J': 0x124E,
	'K': 0x5D6D, 'L': 0x4927, 'M': 0x5FE9, 'N': 0x5F6D, 'O': 0x7B6F,
	'P': 0x7BE4, 'Q': 0x7B7B, 'R': 0x7BED, 'S': 0x79CF, 'T': 0x7492,
	'U': 0x5B6F, 'V': 0x5B6A, 'W': 0x5BFD, 'X': 0x5AAD, 'Y': 0x5A92,
	'Z': 0x72A7,

	// Cyrillic letters shared by Ukrainian/Russian reuse Latin shapes where
	// possible and define compact matching glyphs for the rest.
	'А': 0x2BED, 'Б': 0x79AE, 'В': 0x6BAE, 'Г': 0x7924, 'Ґ': 0x1F24,
	'Д': 0x2B7D, 'Е': 0x79E7, 'Є': 0x39A3, 'Ё': 0x5F3F, 'Ж': 0x55D5,
	'З': 0x72CF, 'И': 0x5F6D, 'І': 0x7497, 'Ї': 0x5497, 'Й': 0x557D,
	'К': 0x5D6D, 'Л': 0x3B6D, 'М': 0x5FE9, 'Н': 0x5BED, 'О': 0x7B6F,
	'П': 0x7B6D, 'Р': 0x7BE4, 'С': 0x7927, 'Т': 0x7492, 'У': 0x5A92,
	'Ф': 0x2F7A, 'Х': 0x5AAD, 'Ц': 0x5B79, 'Ч': 0x5AC9, 'Ш': 0x5B6F,
	'Щ': 0x5BF9, 'Ъ': 0x64D3, 'Ы': 0x5BAE, 'Ь': 0x49AE, 'Э': 0x62CE,
	'Ю': 0x5F6F, 'Я': 0x3AED,

	' ': 0x0000, '-': 0x01C0, '_': 0x0007, '.': 0x0002, ',': 0x000A,
	':': 0x0410, ';': 0x200A, '!': 0x2492, '?': 0x72C2, '/': 0x12A4,
	'\\': 0x4489, '+': 0x05D0, '=': 0x0E38, '(': 0x2488, ')': 0x1112,
	'[': 0x6926, ']': 0x324B, '&': 0x2AAE, '%': 0x5295, '#': 0x5F7D,
	'@': 0x7BE7, '\'': 0x2400, '"': 0x5A00, '>': 0x22A2, '<': 0x1144,
	'№': 0x5F4A,
}

// Cyrillic glyphs are stored as single bytes in runtime menu fields. ASCII
// keeps its byte value; the union of the two Cyrillic alphabets occupies
// 0x80..0xA4 and № uses 0xA5.
var gbaRuntimeCodes = map[rune]byte{
	'А': 0x80, 'Б': 0x81, 'В': 0x82, 'Г': 0x83, 'Ґ': 0x84,
	'Д': 0x85, 'Е': 0x86, 'Є': 0x87, 'Ё': 0x88, 'Ж': 0x89,
	'З': 0x8A, 'И': 0x8B, 'І': 0x8C, 'Ї': 0x8D, 'Й': 0x8E,
	'К': 0x8F, 'Л': 0x90, 'М': 0x91, 'Н': 0x92, 'О': 0x93,
	'П': 0x94, 'Р': 0x95, 'С': 0x96, 'Т': 0x97, 'У': 0x98,
	'Ф': 0x99, 'Х': 0x9A, 'Ц': 0x9B, 'Ч': 0x9C, 'Ш': 0x9D,
	'Щ': 0x9E, 'Ъ': 0x9F, 'Ы': 0xA0, 'Ь': 0xA1, 'Э': 0xA2,
	'Ю': 0xA3, 'Я': 0xA4, '№': 0xA5,
}

func normalizeGBAPunctuation(r rune) string {
	switch r {
	case '’', 'ʼ', '‘', '`':
		return "'"
	case '“', '”', '„':
		return "\""
	case '–', '—', '−':
		return "-"
	case '…':
		return "..."
	default:
		return string(r)
	}
}

func isSupportedGBARune(r rune) bool {
	_, ok := gbaGlyphs[r]
	return ok
}

func normalizeGBAText(value string) (string, []rune) {
	value = strings.ToUpper(value)
	var clean strings.Builder
	unsupported := make([]rune, 0)
	seenUnsupported := map[rune]bool{}
	for _, sourceRune := range value {
		if unicode.IsSpace(sourceRune) {
			clean.WriteByte(' ')
			continue
		}
		normalized := normalizeGBAPunctuation(sourceRune)
		accepted := true
		for _, r := range normalized {
			if isSupportedGBARune(r) {
				clean.WriteRune(r)
			} else {
				accepted = false
			}
		}
		if !accepted {
			clean.WriteByte(' ')
			if !seenUnsupported[sourceRune] {
				seenUnsupported[sourceRune] = true
				unsupported = append(unsupported, sourceRune)
			}
		}
	}
	return strings.Join(strings.Fields(clean.String()), " "), unsupported
}

func truncateGBAText(value string, maximum int) string {
	if maximum <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) > maximum {
		runes = runes[:maximum]
	}
	return strings.TrimSpace(string(runes))
}

func sanitizeGBAText(value string, maximum int) string {
	clean, _ := normalizeGBAText(value)
	return truncateGBAText(clean, maximum)
}

func unsupportedGBARunes(value string) []rune {
	_, unsupported := normalizeGBAText(value)
	return unsupported
}

func gbaTextLength(value string) int {
	return len([]rune(value))
}

func gbaGlyphBits(r rune) uint16 {
	if r >= 'a' && r <= 'z' {
		r -= 'a' - 'A'
	} else if r >= 'а' && r <= 'я' {
		r = unicode.ToUpper(r)
	} else {
		r = unicode.ToUpper(r)
	}
	return gbaGlyphs[r]
}

func encodeGBATextFixed(value string, maximum int) []byte {
	clean := sanitizeGBAText(value, maximum)
	out := make([]byte, maximum)
	for index, r := range []rune(clean) {
		if index >= maximum {
			break
		}
		if r <= 0x7F {
			out[index] = byte(r)
		} else if code, ok := gbaRuntimeCodes[r]; ok {
			out[index] = code
		}
	}
	return out
}

var gbaHeaderTransliteration = map[rune]string{
	'А': "A", 'Б': "B", 'В': "V", 'Г': "G", 'Ґ': "G", 'Д': "D",
	'Е': "E", 'Є': "YE", 'Ё': "YO", 'Ж': "ZH", 'З': "Z", 'И': "I",
	'І': "I", 'Ї': "YI", 'Й': "Y", 'К': "K", 'Л': "L", 'М': "M",
	'Н': "N", 'О': "O", 'П': "P", 'Р': "R", 'С': "S", 'Т': "T",
	'У': "U", 'Ф': "F", 'Х': "H", 'Ц': "TS", 'Ч': "CH", 'Ш': "SH",
	'Щ': "SH", 'Ъ': "", 'Ы': "Y", 'Ь': "", 'Э': "E", 'Ю': "YU", 'Я': "YA",
}

func safeGBAHeaderTitle(value string) []byte {
	value = strings.ToUpper(value)
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '_' || r == '-':
			b.WriteRune(r)
		case gbaHeaderTransliteration[r] != "":
			b.WriteString(gbaHeaderTransliteration[r])
		default:
			if unicode.IsSpace(r) {
				b.WriteByte(' ')
			}
		}
	}
	text := strings.Join(strings.Fields(b.String()), " ")
	if text == "" {
		text = "GBA MEDIA"
	}
	raw := []byte(text)
	if len(raw) > 12 {
		raw = raw[:12]
	}
	out := []byte("            ")
	copy(out, raw)
	return out
}
