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
type Screener struct {
	width        int
	height       int
	screenBuffer []uint32

	textData string // 그릴 문자열
	fgColor  uint32 // 전경색 (문자)
	bgColor  uint32 // 배경색
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

// ReflectText2ScreenBuffer: 매개변수로 text만 받아서,
// Screen 구조체 필드에 있는 (textX, textY, fgColor, bgColor)로 그려준다.
func (s *Screener) ReflectText2ScreenBuffer(text string) {
	x := s.textX
	y := s.textY

	for _, ch := range text {
		glyph, ok := glp.GlyphMap[ch]
		if !ok {
			glyph = glp.Glyph{} // 없는 문자는 빈칸
		}
		s.reflectGlyph(x, y, glyph, s.fgColor, s.bgColor)
		x += glp.GlyphWidth
	}
}

// Draw: 스크린에 텍스트 표시 후 화면 갱신
func (s *Screener) Draw() {
	// 1) 전체 화면 clear
	s.Clear(s.bgColor)

	// 2) 텍스트 그리기 (예: (50, 50)에 Draw)
	s.reflectText(50, 50, s.textData, s.fgColor, s.bgColor)

	// 3) 최종 버퍼 Flush
	s.FlushBuffer()
}

// ReflectCursorAt: 특정 문자 뒤에 커서를 배치
// 예: numAfterText번째 문자 뒤에 커서
func (s *Screener) ReflectCursorAt(numAfterText int) {
	x := s.textX + (numAfterText * glp.GlyphWidth)
	y := s.textY
	s.cursor.ReflectCursor(s, x, y)
}

// ClearCursor: 커서 복원
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

	// Screen 인스턴스 생성
	s := &Screener{
		width:        width,
		height:       height,
		screenBuffer: make([]uint32, width*height),

		textData: " ",        // 예시 텍스트
		fgColor:  0xFF000000, // 검정
		bgColor:  0xFFFFFFFF, // 흰색
		textX:    50,
		textY:    50,
		conn:     conn,
		window:   windowId,
		gc:       gcId,
		depth:    defaultScreen.RootDepth,
		// 커서 생성 (폭=2, 높이=글리프 높이, 검정색)
		cursor: NewCursor(2, glp.GlyphHeight, 0xFF000000),
	}

	return s, nil
}

// ScreenBuffer: screenBuffer getter
func (s *Screener) ScreenBuffer() []uint32 {
	return s.screenBuffer
}

// SetScreenBuffer: screenBuffer 전체를 교체(원한다면)
func (s *Screener) SetScreenBuffer(buf []uint32) {
	if len(buf) == s.width*s.height {
		s.screenBuffer = buf
	}
}

// TextData: 현재 설정된 텍스트 반환
func (s *Screener) TextData() string {
	return s.textData
}

// SetTextData: 텍스트 변경
func (s *Screener) SetTextData(txt string) {
	s.textData = txt
}

// Clear: 전체 화면을 특정 색으로 채우기
func (s *Screener) Clear(color uint32) {
	for i := range s.screenBuffer {
		s.screenBuffer[i] = color
	}
}

// 내부 메서드: 픽셀 세팅
func (s *Screener) setPixel(x, y int, color uint32) {
	if x < 0 || x >= s.width || y < 0 || y >= s.height {
		return
	}
	s.screenBuffer[y*s.width+x] = color
}

// 내부 메서드: 8x8 글리프 그리기
func (s *Screener) reflectGlyph(x, y int, glyph glp.Glyph, fgColor uint32, bgColor uint32) {
	for row := 0; row < glp.GlyphHeight; row++ {
		lineBits := byte(glyph[row])
		for col := 0; col < glp.GlyphWidth; col++ {
			mask := byte(1 << (7 - col))
			if (lineBits & mask) != 0 {
				s.setPixel(x+col, y+row, fgColor)
			} else {
				s.setPixel(x+col, y+row, bgColor)
			}
		}
	}
}

// 내부 메서드: 문자열을 8픽셀씩 나열하여 그리기
func (s *Screener) reflectText(x, y int, text string, fgColor, bgColor uint32) {
	for _, ch := range text {
		glyph, ok := glp.GlyphMap[ch]
		if !ok {
			glyph = glp.Glyph{} // 없는 문자는 빈칸
		}
		s.reflectGlyph(x, y, glyph, fgColor, bgColor)
		x += glp.GlyphWidth
	}
}
func (s *Screener) WaitForEvent() (xgb.Event, xgb.Error) {
	return s.conn.WaitForEvent()
}

// FlushBuffer: 스크린 버퍼 → X 서버로 전송
func (s *Screener) FlushBuffer() {
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
				// ARGB → B, G, R, X
				r := byte((c >> 16) & 0xFF)
				g := byte((c >> 8) & 0xFF)
				b := byte(c & 0xFF)
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
