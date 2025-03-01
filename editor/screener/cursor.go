package screener

// Cursor: 커서를 독립 객체로 정의
type Cursor struct {
	width, height  int      // 커서 크기 (폭, 높이)
	color          uint32   // 커서 색상 (ARGB)
	capturedBuffer []uint32 // 오버라이팅 전, 화면 버퍼 백업용

	currentX, currentY int  // 커서가 마지막으로 그려진 위치
	visible            bool // 현재 커서가 그려져 있는지 여부
}

// NewCursor: 커서 생성자
func NewCursor(width, height int, color uint32) *Cursor {
	return &Cursor{
		width:  width,
		height: height,
		color:  color,
		// 나머지 필드는 0 or nil로 기본값
	}
}

// ReflectCursor: 커서를 Screener에 그린다.
//   - 그리기 전, 화면 버퍼를 capturedBuffer에 백업해 둔다. (부분만)
func (c *Cursor) ReflectCursor(screen *Screener, x, y int) {
	// 1) 이미 커서가 있다면 ClearCursor()로 복원
	if c.visible {
		c.ClearCursor(screen)
	}

	c.currentX = x
	c.currentY = y

	// 2) 오버라이팅할 픽셀 영역(커서 폭*높이)을 백업
	c.captureBuffer(screen)

	// 3) 커서 픽셀로 덮어쓰기
	for row := 0; row < c.height; row++ {
		for col := 0; col < c.width; col++ {
			screen.setPixel(x+col, y+row, c.color)
		}
	}
	c.visible = true
}

// ClearCursor: 캡쳐해 둔 버퍼로 복원
func (c *Cursor) ClearCursor(screen *Screener) {
	if !c.visible {
		return
	}

	// 백업해둔 capturedBuffer를 이용해, 원본 화면 복원
	c.restoreBuffer(screen)

	c.visible = false
}

// captureBuffer: 커서를 그리기 전, 오버라이팅될 영역을 백업
func (c *Cursor) captureBuffer(screen *Screener) {
	c.capturedBuffer = make([]uint32, c.width*c.height)

	idx := 0
	for row := 0; row < c.height; row++ {
		for col := 0; col < c.width; col++ {
			x := c.currentX + col
			y := c.currentY + row
			if x < 0 || x >= screen.width || y < 0 || y >= screen.height {
				c.capturedBuffer[idx] = 0 // 화면 밖은 그냥 0
			} else {
				c.capturedBuffer[idx] = screen.screenBuffer[y*screen.width+x]
			}
			idx++
		}
	}
}

// restoreBuffer: 캡처해둔 버퍼를 원위치에 복원
func (c *Cursor) restoreBuffer(screen *Screener) {
	if c.capturedBuffer == nil {
		return
	}

	idx := 0
	for row := 0; row < c.height; row++ {
		for col := 0; col < c.width; col++ {
			x := c.currentX + col
			y := c.currentY + row
			if x < 0 || x >= screen.width || y < 0 || y >= screen.height {
				idx++
				continue
			}
			screen.screenBuffer[y*screen.width+x] = c.capturedBuffer[idx]
			idx++
		}
	}
}
