package screener

import (
	"fmt"

	glp "go_editor/editor/screener/glyph"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

// Screener 구조체: 내부적으로 화면 버퍼, 텍스트 데이터, X 연결 상태 등을 저장하고
//
//	그걸 렌더링하는 역할 수행
const LineHeight = 16 // 한 줄 높이 16픽셀
type Screener struct {
	width  int
	height int
	// lineCount = height / LineHeight (예시)
	lineCount int

	// screenLines[lineIndex] → []uint32, 길이 = (LineHeight * width)
	// 라인 하나에 16*width 픽셀
	screenLines  [][]uint32
	screenBuffer []uint32

	fgColor uint32 // 전경색 (문자)
	bgColor uint32 // 배경색
	// 텍스트 출력 위치/색상
	textX  int
	textY  int
	conn   *xgb.Conn       // XGB 연결
	window xproto.Window   // 윈도우
	gc     xproto.Gcontext // 그래픽 컨텍스트
	depth  byte            // 디스플레이 비트 깊이
	// 커서 객체 (원하면 여러 개 커서도 가능)
	cursor *Cursor
}

// // Draw: 스크린에 텍스트 표시 후 화면 갱신
// func (s *Screener) Draw() {
// 	// 1) 전체 화면 clear
// 	s.Clear(s.bgColor)

// 	// 2) 텍스트 그리기 (예: (50, 50)에 Draw)
// 	s.reflectText(50, 50, s.textData, s.fgColor, s.bgColor)

// 	// 3) 최종 버퍼 Flush
// 	s.FlushBuffer()
// }

// ReflectCursorAt: lineIndex, charIndex
// cursor.ReflectCursor => lineBuffer 상에 커서 픽셀 덮어쓰기
func (s *Screener) ReflectCursorAt(lineIndex, charIndex int) {
	if lineIndex < 0 || lineIndex >= s.lineCount {
		return
	}
	// 커서를 라인 내부에 그린다
	// x => charIndex*glyphWidth, y => (LineHeight-c.height)/2, ...
	x := charIndex * glp.GlyphWidth
	y := (LineHeight - s.cursor.height) / 2
	s.cursor.ReflectCursor(s, lineIndex, y, x) // lineIndex, row=y, col=x
}

// ClearCursor:
func (s *Screener) ClearCursor() {
	s.cursor.ClearCursor(s)
}

// NewScreener: Screen 생성 + X 윈도우/GC 초기화
func NewScreener(width, height int) (*Screener, error) {
	// X 서버 연결
	conn, err := xgb.NewConn()
	if err != nil {
		return nil, fmt.Errorf("XGB 연결 실패: %v", err)
	}

	setup := xproto.Setup(conn)
	defaultScreen := setup.DefaultScreen(conn)

	// 윈도우 생성
	windowId, err := xproto.NewWindowId(conn)
	if err != nil {
		return nil, err
	}
	xproto.CreateWindow(
		conn,
		xproto.WindowClassCopyFromParent,
		windowId,
		defaultScreen.Root,
		0, 0, // x, y 위치
		uint16(width),  // 폭
		uint16(height), // 높이
		0,              // border width
		xproto.WindowClassInputOutput,
		defaultScreen.RootVisual,
		xproto.CwBackPixel|xproto.CwEventMask,
		[]uint32{
			defaultScreen.WhitePixel,
			xproto.EventMaskExposure | xproto.EventMaskKeyPress,
		},
	)

	// GC 생성
	gcId, err := xproto.NewGcontextId(conn)
	if err != nil {
		return nil, err
	}
	xproto.CreateGC(
		conn,
		gcId,
		xproto.Drawable(windowId),
		xproto.GcForeground|xproto.GcBackground,
		[]uint32{
			defaultScreen.BlackPixel,
			defaultScreen.WhitePixel,
		},
	)

	// 윈도우 맵핑(표시)
	xproto.MapWindow(conn, windowId)
	// 라인 수 계산
	lineCount := height / LineHeight
	screenLines := make([][]uint32, lineCount)
	for i := 0; i < lineCount; i++ {
		// 한 줄 = 16*width
		screenLines[i] = make([]uint32, LineHeight*width)
	}
	// Screen 인스턴스 생성
	s := &Screener{
		width:  width,
		height: height,
		// 라인 초기화 (아직 0개 라인)
		lineCount:   lineCount,
		screenLines: screenLines,
		// 최종 flush 시 사용
		screenBuffer: make([]uint32, width*height),
		fgColor:      0xFF000000, // 검정
		bgColor:      0xFFFFFFFF, // 흰색
		textX:        50,
		textY:        50,
		conn:         conn,
		window:       windowId,
		gc:           gcId,
		depth:        defaultScreen.RootDepth,
		// 커서 생성 (폭=2, 높이=글리프 높이, 검정색)
		cursor: NewCursor(2, glp.GlyphHeight, 0xFF000000),
	}

	return s, nil
}

// ReflectLine: lineIndex번째 라인에 text를 중앙(수직) 기준으로 그린다.
// 1) 라인 전체를 bgColor로 초기화
// 2) 왼쪽 x=0부터 글리프 찍기
// 3) 수직 위치 => (LineHeight - GlyphHeight)/2
func (s *Screener) ReflectLine(lineIndex int, text string) {
	if lineIndex < 0 || lineIndex >= s.lineCount {
		return
	}
	linePixels := s.screenLines[lineIndex]
	// 1) 배경색으로 초기화
	for i := range linePixels {
		linePixels[i] = s.bgColor
	}

	// 2) 글자 그리기
	drawX := 0
	yOffset := (LineHeight - glp.GlyphHeight) / 2 // 수직 중앙
	for _, ch := range text {
		glyph, ok := glp.GlyphMap[ch]
		if !ok {
			glyph = glp.Glyph{}
		}
		s.drawGlyphToLine(linePixels, drawX, yOffset, glyph, s.fgColor)
		drawX += glp.GlyphWidth
		if drawX >= s.width {
			break
		}
	}
	fmt.Printf("[Screener] ReflectLine(%d): %q\n", lineIndex, text)
}

// drawGlyphToLine: 한 줄(16*width)의 픽셀에 글리프를 배치
func (s *Screener) drawGlyphToLine(linePixels []uint32, startX, startY int, glyph glp.Glyph, fg uint32) {
	// linePixels는 높이=16, 폭=width
	for row := 0; row < glp.GlyphHeight; row++ {
		lineBits := glyph[row]
		for col := 0; col < glp.GlyphWidth; col++ {
			mask := byte(1 << (7 - col))
			if (byte(lineBits) & mask) != 0 {
				px := startX + col
				py := startY + row
				if px < 0 || px >= s.width {
					continue
				}
				if py < 0 || py >= LineHeight {
					continue
				}
				// index in linePixels = py*width + px
				idx := py*s.width + px
				linePixels[idx] = fg
			}
		}
	}
}

// Clear: 모든 라인을 bgColor로 초기화
func (s *Screener) Clear(color uint32) {
	fmt.Printf("클리어 시작")
	for i := 0; i < s.lineCount; i++ {
		linePixels := s.screenLines[i]
		for px := range linePixels {
			linePixels[px] = color
		}
	}
	fmt.Printf("[Screener] Clear => 0x%X\n", color)
}

// setLinePixel: lineIndex 안의 (row, col)에 color 세팅
// row in [0..LineHeight-1], col in [0..width-1]
func (s *Screener) setLinePixel(lineIndex, row, col int, color uint32) {
	if lineIndex < 0 || lineIndex >= s.lineCount {
		return
	}
	if row < 0 || row >= LineHeight {
		return
	}
	if col < 0 || col >= s.width {
		return
	}
	idx := row*s.width + col
	s.screenLines[lineIndex][idx] = color
}

func (s *Screener) WaitForEvent() (xgb.Event, xgb.Error) {
	return s.conn.WaitForEvent()
}

// FlushBuffer: line기준 => 전체 스크린 버퍼 => X 서버
func (s *Screener) FlushBuffer() {
	// (1) line들을 하나의 screenBuffer로 합침
	// lineIndex=0 => y=0..15
	// lineIndex=1 => y=16..31

	for lineIndex := 0; lineIndex < s.lineCount; lineIndex++ {
		linePixels := s.screenLines[lineIndex]
		// 한 줄(16행 * width열)
		for row := 0; row < LineHeight; row++ {
			for col := 0; col < s.width; col++ {
				pix := linePixels[row*s.width+col]
				y := lineIndex*LineHeight + row
				x := col
				s.screenBuffer[y*s.width+x] = pix
			}
		}
	}

	// (2) 기존 chunkHeight=64로 전송
	chunkHeight := 64
	for yStart := 0; yStart < s.height; yStart += chunkHeight {
		h := chunkHeight
		if yStart+h > s.height {
			h = s.height - yStart
		}
		data := make([]byte, s.width*h*4)
		idx := 0
		for row := yStart; row < yStart+h; row++ {
			for col := 0; col < s.width; col++ {
				c := s.screenBuffer[row*s.width+col]
				r := byte((c >> 16) & 0xFF)
				g := byte((c >> 8) & 0xFF)
				b := byte(c & 0xFF)
				// ARGB => B,G,R,X
				data[idx+0] = b
				data[idx+1] = g
				data[idx+2] = r
				data[idx+3] = 0
				idx += 4
			}
		}
		xproto.PutImage(
			s.conn,
			xproto.ImageFormatZPixmap,
			xproto.Drawable(s.window),
			s.gc,
			uint16(s.width),
			uint16(h),
			0, int16(yStart),
			0,
			s.depth,
			data,
		)
	}
}
