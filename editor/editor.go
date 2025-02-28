package editor

import (
	"fmt"
	"time"

	"go_editor/editor/screener"

	"github.com/BurntSushi/xgb/xproto"
)

// Editor: screener를 가지고,
//
//	FPS 기반 화면 업데이트 & 이벤트 루프를 관리
type Editor struct {
	screen    *screener.Screener
	fpsTicker *time.Ticker
	running   bool
	textCount int
}

// NewEditor: Editor 인스턴스 생성
func NewEditor(width, height int, fps int) (*Editor, error) {
	scr, err := screener.NewScreener(width, height)
	if err != nil {
		return nil, err
	}

	e := &Editor{
		screen:    scr,
		fpsTicker: time.NewTicker(time.Second / time.Duration(fps)),
		running:   true,
		textCount: 0,
	}
	return e, nil
}

// Run: 메인 이벤트 루프
func (e *Editor) Run() {
	for e.running {
		select {
		case <-e.fpsTicker.C:
			// FPS마다 Flush
			e.screen.FlushBuffer()

		default:
			// 이벤트 처리
			ev, err := e.screen.WaitForEvent()
			if err != nil {
				fmt.Println("Error:", err)
				e.running = false
				break
			}
			if ev == nil {
				// 이벤트 없으면 계속 루프
				continue
			}

			switch event := ev.(type) {
			case xproto.ExposeEvent:
				// 노출 시 별도 처리?
				_ = event
			case xproto.KeyPressEvent:
				e.textCount++
				// 새로운 텍스트를 버퍼에 반영
				e.screen.Clear(0xFFFFFFFF)
				e.screen.ReflectText2ScreenBuffer(
					fmt.Sprintf("KeyPress Count: %d", e.textCount),
				)
			}
		}
	}
}

// Stop: Editor 종료
func (e *Editor) Stop() {
	e.running = false
	e.fpsTicker.Stop()
}
