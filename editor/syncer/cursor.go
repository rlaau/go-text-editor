package main

// Cursor: line-based로 커서를 저장
type Cursor struct {
	width, height  int
	color          uint32
	capturedBuffer []uint32

	currentLineBuffer  *LineBuffer
	currentY, currentX int
	visible            bool
}

func NewCursor(width, height int, color uint32) *Cursor {
	return &Cursor{
		width:  width,
		height: height,
		color:  color,
	}
}

// ReflectCursor: 커서를 (lineIndex, row, col)에 그린다
func (c *Cursor) ReflectCursor(sp *SyncProtocol, lineBuffer *LineBuffer, row, col int) {
	if c.visible {
		c.ClearCursor(sp)
	}
	c.currentLineBuffer = lineBuffer
	c.currentY = row
	c.currentX = col

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

	idx := 0
	for ry := 0; ry < c.height; ry++ {
		for cx := 0; cx < c.width; cx++ {
			lineBuffer := c.currentLineBuffer
			r := c.currentY + ry
			cx2 := c.currentX + cx
			if r < 0 || r >= LineHeight {
				c.capturedBuffer[idx] = 0
			} else if cx2 < 0 || cx2 >= sp.screenWidth {
				c.capturedBuffer[idx] = 0
			} else {
				c.capturedBuffer[idx] = lineBuffer.data[r*sp.screenWidth+cx2]
			}
			idx++
		}
	}
}

func (c *Cursor) restoreBuffer(sp *SyncProtocol) {
	if c.capturedBuffer == nil {
		return
	}
	idx := 0
	for ry := 0; ry < c.height; ry++ {
		for cx := 0; cx < c.width; cx++ {
			lineBuffer := c.currentLineBuffer
			r := c.currentY + ry
			cx2 := c.currentX + cx

			if r < 0 || r >= LineHeight {
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
