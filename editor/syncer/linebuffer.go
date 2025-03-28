package syncer

import (
	glp "go_editor/editor/screener/glyph"
)

type LineBuffer struct {
	data []uint32
}

func (sp *SyncProtocol) NewLineBuffer() *LineBuffer {

	data := make([]uint32, sp.LineHeight*sp.screenWidth)

	return &LineBuffer{data: data}
}

// FlushLineBuffer: collects all LineBuffer data from nodes into a 2D array for rendering
func (sp *SyncProtocol) FlushLineBuffer() [][]uint32 {

	lineBuffers := [][]uint32{}

	if sp.IsCursorVisible() {
		sp.cursor.CusorDrawOn(sp)
	} else {
		sp.cursor.ClearCursor(sp)
	}

	head := sp.syncData.head
	if head == nil {
		println("emtpy list")
		return lineBuffers
	}

	// Check if syncData exists and has nodes
	if sp.syncData != nil && sp.syncData.head != nil {
		// Iterate through all nodes in the linked list
		sp.syncData.ForEach(func(node *SyncNode) {
			if node.LineBuffer != nil {
				// Create a copy of the line data to avoid reference issues
				lineData := make([]uint32, len(node.LineBuffer.data))
				copy(lineData, node.LineBuffer.data)

				// Add this line to our collection
				lineBuffers = append(lineBuffers, lineData)
			}
		})
	}

	return lineBuffers
}

func (sp *SyncProtocol) ReflectLine(l *LineBuffer, text string) {
	//배경색 칠하기
	for i := range l.data {
		l.data[i] = sp.bgColor
	}
	drawX := 0
	yOffset := (sp.LineHeight - glp.GlyphHeight) / 2 // 수직 중앙
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
				if py < 0 || py >= sp.LineHeight {
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

	if row < 0 || row >= sp.LineHeight {
		return
	}
	if col < 0 || col >= sp.screenWidth {
		return
	}
	idx := row*sp.screenWidth + col
	lineBuffer.data[idx] = color
}
