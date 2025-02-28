package main

type Width byte
type Height byte

const (
	GlyphHeight = 8
	GlyphWidth  = 8
)

type Glyph [GlyphHeight]Width

// 최종 글리프 맵
var GlyphMap = map[rune]Glyph{}

// ✅ `init()` 함수: 모든 글리프를 `glyphMap`에 추가
func init() {
	for char, glyph := range NumberGlyphMap {
		GlyphMap[char] = glyph
	}
	for char, glyph := range LowercaseGlyphMap {
		GlyphMap[char] = glyph
	}
	for char, glyph := range UppercaseGlyphMap {
		GlyphMap[char] = glyph
	}
	for char, glyph := range SpecialGlyphMap {
		GlyphMap[char] = glyph
	}
}
