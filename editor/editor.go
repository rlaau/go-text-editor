package editor

import (
	"fmt"
	"time"

	"go_editor/editor/screener"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

//TODO 텍스트의 관리를 Screener에서 뺏어오기
//TODO 스크리너는 걍 텍스트 "받아서" 그 인자 바탕으로 렌더링만 구현
//TODO 그 텍스트를 주고, 또 메니징 하는 것은 에디터가 하기. rope구조 등으로

// Editor: screener를 가지고,
//
//	FPS 기반 화면 업데이트 & 이벤트 루프를 관리// Editor: screener를 가지고, FPS 기반 화면 업데이트 + 커서 깜빡임 + 이벤트 처리
type Editor struct {
	screen      *screener.Screener
	fpsTicker   *time.Ticker // 30FPS
	blinkTicker *time.Ticker // 1초 주기 커서 깜빡
	running     bool
	textCount   int

	cursorVisible bool
	cursorPos     int // "5번째" 글자 뒤에 커서

	eventChan chan xgb.Event
}

// NewEditor: Editor 인스턴스 생성
func NewEditor(width, height int, fps int) (*Editor, error) {
	scr, err := screener.NewScreener(width, height)
	if err != nil {
		return nil, err
	}

	e := &Editor{
		screen:        scr,
		fpsTicker:     time.NewTicker(time.Second / time.Duration(fps)), // 30FPS
		blinkTicker:   time.NewTicker(time.Second * 1),                  // 1초 주기
		running:       true,
		textCount:     0,
		cursorVisible: false,
		cursorPos:     3,                        // 5번째 글자 뒤
		eventChan:     make(chan xgb.Event, 20), // 이벤트 버퍼
	}

	return e, nil
}

// collectEvents: 별도 고루틴에서 X 이벤트를 무한정 수신하여 eventChan에 보냄
func (e *Editor) collectEvents() {
	for {
		ev, err := e.screen.WaitForEvent() // 블로킹
		if err != nil {
			// 에러 발생 시 채널 닫고 종료
			close(e.eventChan)
			return
		}
		if ev != nil {
			// 이벤트를 eventChan으로 전달
			e.eventChan <- ev
		}
	}
}

// Run: 메인 이벤트 루프
func (e *Editor) Run() {
	// 이벤트 전용 고루틴: 블로킹 WaitForEvent() → eventChan 로 전달
	e.screen.Clear(0xFFFFFFFF)
	go e.collectEvents()

	for e.running {
		select {
		case <-e.blinkTicker.C:
			// 1초마다 커서 깜빡
			e.toggleCursorBlink()

		case <-e.fpsTicker.C:
			// 30FPS로 화면 Flush
			e.screen.FlushBuffer()

		case ev, ok := <-e.eventChan:
			// 이벤트 채널에서 이벤트 수신
			if !ok {
				// 채널이 닫힘 => 종료
				e.running = false
				break
			}
			// 이벤트 처리
			switch ev.(type) {
			case xproto.ExposeEvent:
				// 노출 이벤트 (원한다면 화면 다시 그려도 됨)
				// e.screen.Clear(0xFFFFFFFF)
				// e.screen.ReflectText2ScreenBuffer(fmt.Sprintf("KeyPress Count: %d", e.textCount))
				// if e.cursorVisible {
				// 	e.screen.ReflectCursorAt(e.cursorPos)
				// }
			case xproto.KeyPressEvent:
				// 키 입력 => textCount++
				e.textCount++
				e.screen.Clear(0xFFFFFFFF)
				e.screen.ReflectText2ScreenBuffer(
					fmt.Sprintf("KeyPress Count: %d", e.textCount),
				)
				if e.cursorVisible {
					e.screen.ReflectCursorAt(e.cursorPos)
				}
			}
		}
	}
}

// Stop: Editor 종료
func (e *Editor) Stop() {
	e.running = false
	e.fpsTicker.Stop()
	e.blinkTicker.Stop()
}

// toggleCursorBlink: 커서 깜빡임 토글
func (e *Editor) toggleCursorBlink() {
	if e.cursorVisible {
		// 이미 커서 있으면 지움
		e.screen.ClearCursor()
		e.cursorVisible = false
	} else {
		// 없으면 그려줌
		e.screen.ReflectCursorAt(e.cursorPos)
		e.cursorVisible = true
	}
}
