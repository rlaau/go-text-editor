package syncer

import glp "go_editor/editor/screener/glyph"

// Cursor: line-based로 커서를 저장
type Cursor struct {
	width, height  int
	color          uint32
	capturedBuffer []uint32

	currentLineBuffer *LineBuffer
	currentCharInset  int

	visible bool
}

func (sp *SyncProtocol) IsCursorVisible() bool {
	return sp.cursor.visible
}
func (sp *SyncProtocol) SetCursorVisible(visible bool) {
	sp.cursor.visible = visible
}
func (sp *SyncProtocol) CursorDrawOn() {
	sp.cursor.CusorDrawOn(sp)
}

// ClearCursor:
func (sp *SyncProtocol) ClearCursor() {
	sp.cursor.ClearCursor(sp)
}

func NewCursor(width, height int, color uint32) *Cursor {
	return &Cursor{
		width:  width,
		height: height,
		color:  color,
	}
}

func (c *Cursor) GetCoordinate() (*LineBuffer, int) {
	return c.currentLineBuffer, c.currentCharInset
}
func (c *Cursor) CusorDrawOn(sp *SyncProtocol) {
	c.CoordinateCursor(sp, c.currentLineBuffer, c.currentCharInset)

}

// ReflectCursorAt: lineIndex, charIndex
// cursor.ReflectCursor => lineBuffer 상에 커서 픽셀 덮어쓰기
func (sp *SyncProtocol) ReflectCursorAt(lineBuffer *LineBuffer, charInset int) {

	sp.cursor.CoordinateCursor(sp, lineBuffer, charInset) // lineIndex, row=y, col=x
}

// CoordinateCursor: 커서를 (lineIndex, row, col)에 그린다
func (c *Cursor) CoordinateCursor(sp *SyncProtocol, lineBuffer *LineBuffer, charInset int) {

	c.ClearCursor(sp)
	//우선 c의 상태를 변경
	c.currentLineBuffer = lineBuffer
	c.currentCharInset = charInset
	//변경된 c의 상태 바탕으로 픽셀 데이터에 매핑

	col, row := c.mapInset2pixColRow(sp)

	c.captureBuffer(sp)
	// 커서 폭*높이만큼 픽셀 덮어쓰기
	for ry := 0; ry < c.height; ry++ {
		for cx := 0; cx < c.width; cx++ {
			//이것도 결국 리플렉트임. 해당 라인버퍼에 그리는 거여서
			sp.setLinePixel(lineBuffer, row+ry, col+cx, c.color)
		}
	}
	c.visible = true
}

// mapInset2pixXY는 커서의 currentCharInset을 바탕으로, 커서의 좌상단 픽셀의 좌표를 리턴한다.
// 즉, 커서의 인셋을 바탕으로 픽셀상의 스타팅 포인트 제공
func (c *Cursor) mapInset2pixColRow(sp *SyncProtocol) (col int, row int) {
	col = c.currentCharInset * glp.GlyphWidth
	row = (sp.LineHeight - sp.cursor.height) / 2
	return col, row

}

// ClearCursor: 복원
func (c *Cursor) ClearCursor(sp *SyncProtocol) {
	if !c.visible {
		return
	}
	c.restoreBuffer(sp)
	c.visible = false
}

// captureBuffer: 덮어쓸 영역 백업
func (c *Cursor) captureBuffer(sp *SyncProtocol) {
	c.capturedBuffer = make([]uint32, c.width*c.height)
	startCol, startRow := c.mapInset2pixColRow(sp)

	idx := 0
	for ry := 0; ry < c.height; ry++ {
		for cx := 0; cx < c.width; cx++ {
			lineBuffer := c.currentLineBuffer

			ry2 := startRow + ry
			cx2 := startCol + cx
			if ry2 < 0 || ry2 >= sp.LineHeight {
				c.capturedBuffer[idx] = 0
			} else if cx2 < 0 || cx2 >= sp.screenWidth {
				c.capturedBuffer[idx] = 0
			} else {
				c.capturedBuffer[idx] = lineBuffer.data[ry2*sp.screenWidth+cx2]
			}
			idx++
		}
	}
}

func (c *Cursor) restoreBuffer(sp *SyncProtocol) {
	if c.capturedBuffer == nil {
		return
	}
	startCol, startRow := c.mapInset2pixColRow(sp)
	idx := 0
	for ry := 0; ry < c.height; ry++ {
		for cx := 0; cx < c.width; cx++ {
			lineBuffer := c.currentLineBuffer
			r := startRow + ry
			cx2 := startCol + cx

			if r < 0 || r >= sp.LineHeight {
				idx++
				continue
			}
			if cx2 < 0 || cx2 >= sp.screenWidth {
				idx++
				continue
			}
			lineBuffer.data[r*sp.screenWidth+cx2] = c.capturedBuffer[idx]
			idx++
		}
	}
}
