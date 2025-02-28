package main

import (
	"fmt"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

const (
	screenWidth  = 800
	screenHeight = 600
)

var screenBuffer = make([]uint32, screenWidth*screenHeight)

// 픽셀 세팅 (ARGB 가정)
func setPixel(x, y int, color uint32) {
	if x < 0 || x >= screenWidth || y < 0 || y >= screenHeight {
		return
	}
	screenBuffer[y*screenWidth+x] = color
}

// 8x8 그리기
func drawGlyph(x, y int, glyph Glyph, fgColor uint32, bgColor uint32) {
	for row := 0; row < GlyphHeight; row++ {
		lineBits := byte(glyph[row])
		for col := 0; col < GlyphWidth; col++ {
			mask := byte(1 << (7 - col))
			if (lineBits & mask) != 0 {
				setPixel(x+col, y+row, fgColor)
			} else {
				setPixel(x+col, y+row, bgColor)
			}
		}
	}
}

// 문자열을 8픽셀씩 나열
func drawText(x, y int, text string, fgColor, bgColor uint32) {
	for _, ch := range text {
		glyph, ok := GlyphMap[ch]
		if !ok {
			glyph = Glyph{} // 없는 문자는 빈칸
		}
		drawGlyph(x, y, glyph, fgColor, bgColor)
		x += GlyphWidth
	}
}

func flushBuffer(conn *xgb.Conn, window xproto.Window, gc xproto.Gcontext, depth byte) {
	var chunkHeight = 64 // 이 값은 환경에 맞춰 튜닝

	for yStart := 0; yStart < screenHeight; yStart += chunkHeight {
		h := chunkHeight
		if yStart+h > screenHeight {
			h = screenHeight - yStart
		}

		// 이 구역(0, yStart) ~ (screenWidth, h) 크기만큼의 픽셀을 추출
		data := make([]byte, screenWidth*h*4)
		idx := 0
		for row := yStart; row < yStart+h; row++ {
			for col := 0; col < screenWidth; col++ {
				c := screenBuffer[row*screenWidth+col]
				// ARGB -> B, G, R, X 변환
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
			conn,
			xproto.ImageFormatZPixmap,
			xproto.Drawable(window),
			gc,
			uint16(screenWidth),
			uint16(h),
			0, int16(yStart), // ← y offset 위치
			0,
			depth,
			data,
		)
	}
}

func main() {
	conn, err := xgb.NewConn()
	if err != nil {
		panic(err)
	}
	setup := xproto.Setup(conn)
	defaultScreen := setup.DefaultScreen(conn)

	// 윈도우 생성
	windowId, _ := xproto.NewWindowId(conn)
	xproto.CreateWindow(
		conn,
		xproto.WindowClassCopyFromParent,
		windowId,
		defaultScreen.Root,
		0, 0, 800, 600,
		0,
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
		panic(err)
	}
	// !! Drawable(windowId) 로 캐스팅 + GcForeground/GcBackground 상수 사용 !!
	xproto.CreateGC(
		conn,
		gcId,
		xproto.Drawable(windowId),
		xproto.GcForeground|xproto.GcBackground,
		[]uint32{defaultScreen.BlackPixel, defaultScreen.WhitePixel},
	)

	xproto.MapWindow(conn, windowId)

	// 이벤트 루프
	for {
		e, err := conn.WaitForEvent()
		if err != nil {
			fmt.Println("Error:", err)
			break
		}
		switch e.(type) {
		case xproto.ExposeEvent:
			// 노출 시점에 화면 clear 후 "apple!" 텍스트를 그림
			for i := range screenBuffer {
				screenBuffer[i] = 0xFFFFFFFF // 흰색
			}
			drawText(50, 50, "hello world!", 0xFF000000, 0xFFFFFFFF) // 검정/흰색
			flushBuffer(conn, windowId, gcId, defaultScreen.RootDepth)

		case xproto.KeyPressEvent:
			// 지금은 무시
		}
	}
}
