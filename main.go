package main

import (
	"fmt"
	"time"

	scrn "go_editor/screen"

	"github.com/BurntSushi/xgb/xproto"
)

func main() {
	screen, err := scrn.NewScreen(800, 600)
	if err != nil {
		panic(err)
	}

	// 10FPS 주기로 화면 Flush
	ticker := time.NewTicker(time.Second / 10)
	defer ticker.Stop()

	running := true
	textCount := 0

	for running {
		select {
		case <-ticker.C:

			screen.FlushBuffer()

		default:
			// 이벤트 처리 (Non-blocking 예시)
			e, err := screen.WaitForEvent()
			if err != nil {
				// 에러 발생 시 종료
				fmt.Println("Error:", err)
				running = false
				break
			}
			if e != nil {
				switch e.(type) {
				case xproto.ExposeEvent:
					// 노출 시 => 굳이 지금은 아무것도 안 해도 됨
				case xproto.KeyPressEvent:
					// 키 누를 때마다 textCount 증가 & 그 텍스트를 버퍼에 반영
					textCount++
					screen.Clear(0xFFFFFFFF)
					screen.ReflectText2ScreenBuffer(
						fmt.Sprintf("KeyPress | Count: %d", textCount),
					)
				}
			}
		}
	}
}
