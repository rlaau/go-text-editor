package screener

// Cursor: line-based로 커서를 저장
type Cursor struct {
	width, height  int
	color          uint32
	capturedBuffer []uint32

	currentLine, currentRow, currentCol int
	visible                             bool
}

func NewCursor(width, height int, color uint32) *Cursor {
	return &Cursor{
		width:  width,
		height: height,
		color:  color,
	}
}

// ReflectCursor: 커서를 (lineIndex, row, col)에 그린다
func (c *Cursor) ReflectCursor(screen *Screener, lineIndex, row, col int) {
	if c.visible {
		c.ClearCursor(screen)
	}
	c.currentLine = lineIndex
	c.currentRow = row
	c.currentCol = col

	c.captureBuffer(screen)
	// 커서 폭*높이만큼 픽셀 덮어쓰기
	for ry := 0; ry < c.height; ry++ {
		for cx := 0; cx < c.width; cx++ {
			screen.setLinePixel(lineIndex, row+ry, col+cx, c.color)
		}
	}
	c.visible = true
}

// ClearCursor: 복원
func (c *Cursor) ClearCursor(screen *Screener) {
	if !c.visible {
		return
	}
	c.restoreBuffer(screen)
	c.visible = false
}

// captureBuffer: 덮어쓸 영역 백업
func (c *Cursor) captureBuffer(screen *Screener) {
	c.capturedBuffer = make([]uint32, c.width*c.height)

	idx := 0
	for ry := 0; ry < c.height; ry++ {
		for cx := 0; cx < c.width; cx++ {
			lineIndex := c.currentLine
			r := c.currentRow + ry
			cx2 := c.currentCol + cx
			if lineIndex < 0 || lineIndex >= screen.lineCount {
				c.capturedBuffer[idx] = 0
			} else if r < 0 || r >= LineHeight {
				c.capturedBuffer[idx] = 0
			} else if cx2 < 0 || cx2 >= screen.width {
				c.capturedBuffer[idx] = 0
			} else {
				c.capturedBuffer[idx] = screen.screenLines[lineIndex][r*screen.width+cx2]
			}
			idx++
		}
	}
}

func (c *Cursor) restoreBuffer(screen *Screener) {
	if c.capturedBuffer == nil {
		return
	}
	idx := 0
	for ry := 0; ry < c.height; ry++ {
		for cx := 0; cx < c.width; cx++ {
			lineIndex := c.currentLine
			r := c.currentRow + ry
			cx2 := c.currentCol + cx
			if lineIndex < 0 || lineIndex >= screen.lineCount {
				idx++
				continue
			}
			if r < 0 || r >= LineHeight {
				idx++
				continue
			}
			if cx2 < 0 || cx2 >= screen.width {
				idx++
				continue
			}
			screen.screenLines[lineIndex][r*screen.width+cx2] = c.capturedBuffer[idx]
			idx++
		}
	}
}
