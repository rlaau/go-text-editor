package screener

import (
	"fmt"

	glp "go_editor/editor/screener/glyph"

	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgbutil"
)

// Screener 구조체: 내부적으로 화면 버퍼, 텍스트 데이터, X 연결 상태 등을 저장하고
//
//	그걸 렌더링하는 역할 수행
const LineHeight = 16 // 한 줄 높이 16픽셀
type Screener struct {
	width        int
	height       int
	lineCount    int
	screenLines  [][]uint32
	screenBuffer []uint32

	fgColor uint32
	bgColor uint32

	textX  int
	textY  int
	xu     *xgbutil.XUtil // XGBUtil 연결 객체
	window xproto.Window
	gc     xproto.Gcontext
	depth  byte
	cursor *Cursor
}

// ReflectCursorAt: lineIndex, charIndex
// cursor.ReflectCursor => lineBuffer 상에 커서 픽셀 덮어쓰기
func (s *Screener) ReflectCursorAt(lineIndex, charIndex int) {
	println("커서 드로우", lineIndex, charIndex)
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

// NewScreener: XGBUtil을 기반으로 Screener 초기화
func NewScreener(xu *xgbutil.XUtil, width, height int, fg, bg uint32) (*Screener, error) {
	setup := xproto.Setup(xu.Conn())
	defaultScreen := setup.DefaultScreen(xu.Conn())

	windowId, err := xproto.NewWindowId(xu.Conn())
	if err != nil {
		return nil, err
	}

	xproto.CreateWindow(
		xu.Conn(),
		xproto.WindowClassCopyFromParent,
		windowId,
		defaultScreen.Root,
		0, 0,
		uint16(width),
		uint16(height),
		0,
		xproto.WindowClassInputOutput,
		defaultScreen.RootVisual,
		xproto.CwBackPixel|xproto.CwEventMask,
		[]uint32{
			defaultScreen.WhitePixel,
			xproto.EventMaskExposure | xproto.EventMaskKeyPress,
		},
	)

	gcId, err := xproto.NewGcontextId(xu.Conn())
	if err != nil {
		return nil, err
	}

	xproto.CreateGC(
		xu.Conn(),
		gcId,
		xproto.Drawable(windowId),
		xproto.GcForeground|xproto.GcBackground,
		[]uint32{
			defaultScreen.BlackPixel,
			defaultScreen.WhitePixel,
		},
	)

	xproto.MapWindow(xu.Conn(), windowId)

	lineCount := height / LineHeight
	screenLines := make([][]uint32, lineCount)
	for i := 0; i < lineCount; i++ {
		screenLines[i] = make([]uint32, LineHeight*width)
	}

	s := &Screener{
		width:        width,
		height:       height,
		lineCount:    lineCount,
		screenLines:  screenLines,
		screenBuffer: make([]uint32, width*height),
		fgColor:      fg,
		bgColor:      bg,
		textX:        50,
		textY:        50,
		xu:           xu,
		window:       windowId,
		gc:           gcId,
		depth:        defaultScreen.RootDepth,
		cursor:       NewCursor(2, glp.GlyphHeight, 0xFF000000),
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

// TODO 여기서부턴 스크리너 고유영역
// TODO XGB나 XU다루는 순간은 스크리너에서 처리

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
			s.xu.Conn(),
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
