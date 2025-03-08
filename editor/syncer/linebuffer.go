package main

import (
	"fmt"
	glp "go_editor/editor/screener/glyph"
)

const LineHeight = 16 // 한 줄 높이 16픽셀
type LineBuffer struct {
	data []uint32
}

func (sp *SyncProtocol) NewLineBuffer() *LineBuffer {
	data := make([]uint32, LineHeight*sp.screenWidth)
	return &LineBuffer{data: data}
}

// ReflectCursorAt: lineIndex, charIndex
// cursor.ReflectCursor => lineBuffer 상에 커서 픽셀 덮어쓰기
func (sp *SyncProtocol) ReflectCursorAt(lineBuffer *LineBuffer, charIndex int) {
	println("커서 드로우 글자위치", charIndex)
	// 커서를 라인 내부에 그린다
	// x => charIndex*glyphWidth, y => (LineHeight-c.height)/2, ...
	x := charIndex * glp.GlyphWidth
	y := (LineHeight - sp.cursor.height) / 2
	sp.cursor.ReflectCursor(sp, lineBuffer, y, x) // lineIndex, row=y, col=x
}

// ClearCursor:
func (sp *SyncProtocol) ClearCursor() {
	sp.cursor.ClearCursor(sp)
}
func (sp *SyncProtocol) ReflectLine(l *LineBuffer, text string) {
	for i := range l.data {
		l.data[i] = sp.bgColor
	}
	drawX := 0
	yOffset := (LineHeight - glp.GlyphHeight) / 2 // 수직 중앙
	for _, ch := range text {
		glyph, ok := glp.GlyphMap[ch]
		if !ok {
			glyph = glp.Glyph{}
		}
		sp.drawGlyphToLine(l, drawX, yOffset, glyph, sp.fgColor)
		drawX += glp.GlyphWidth
		if drawX >= sp.screenWidth {
			break
		}
	}
	fmt.Printf("[SP] ReflectLine: %q\n", text)
}

// drawGlyphToLine: 한 줄(16*width)의 픽셀에 글리프를 배치
func (sp *SyncProtocol) drawGlyphToLine(l *LineBuffer, startX, startY int, glyph glp.Glyph, fg uint32) {
	// linePixels는 높이=16, 폭=width
	for row := 0; row < glp.GlyphHeight; row++ {
		lineBits := glyph[row]
		for col := 0; col < glp.GlyphWidth; col++ {
			mask := byte(1 << (7 - col))
			if (byte(lineBits) & mask) != 0 {
				px := startX + col
				py := startY + row
				if px < 0 || px >= sp.screenWidth {
					continue
				}
				if py < 0 || py >= LineHeight {
					continue
				}
				// index in linePixels = py*width + px
				idx := py*sp.screenWidth + px
				l.data[idx] = fg
			}
		}
	}
}

// setLinePixel: lineIndex 안의 (row, col)에 color 세팅
// row in [0..LineHeight-1], col in [0..width-1]
func (sp *SyncProtocol) setLinePixel(lineBuffer *LineBuffer, row, col int, color uint32) {

	if row < 0 || row >= LineHeight {
		return
	}
	if col < 0 || col >= sp.screenWidth {
		return
	}
	idx := row*sp.screenWidth + col
	lineBuffer.data[idx] = color
}
