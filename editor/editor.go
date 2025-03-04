package editor

import (
	"fmt"

	"go_editor/editor/screener"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

//TODO 텍스트의 관리를 Screener에서 뺏어오기
//TODO 스크리너는 걍 텍스트 "받아서" 그 인자 바탕으로 렌더링만 구현
//TODO 그 텍스트를 주고, 또 메니징 하는 것은 에디터가 하기. rope구조 등으로

//TODO 이후, "명령어"는 커멘더에 위임해서 처리해보기. xgb의 이벤트를 래핑해서 인터프리트

// Editor: screener를 가지고,
//
//	FPS 기반 화면 업데이트 & 이벤트 루프를 관리
//
// Editor: screener를 가지고, FPS 기반 화면 업데이트 + 커서 깜빡임 + 이벤트 처리
type Editor struct {
	screen      *screener.Screener
	fpsTicker   *time.Ticker // 30FPS
	blinkTicker *time.Ticker // 1초 주기 커서 깜빡
	running     bool

	// 간단히 2줄만 관리 (Line 0: "Hello", Line 1: "KeyPress Count: X")
	lines []string

	textCount int

	// 커서 표시
	cursorVisible bool
	cursorLine    int
	cursorChar    int

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
		lines:         []string{"Hello", "KeyPress Count: 0"}, // 초기 2개 라인,
		textCount:     0,
		cursorVisible: false,
		// 커서는 line=1, char=3 초기값
		cursorLine: 1,
		cursorChar: 3,
		eventChan:  make(chan xgb.Event, 20), // 이벤트 버퍼
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
				// line2 수정
				e.lines[1] = fmt.Sprintf("KeyPress Count: %d", e.textCount)

				// 화면 다시 그림
				e.redrawAll()

			}
		}
	}
}

// redrawAll: 모든 라인을 스크리너에 반영
func (e *Editor) redrawAll() {
	e.screen.Clear(0xFFFFFFFF)

	// 간단: line 0 -> y=50, line1 -> y= 70
	for i, text := range e.lines {
		e.screen.ReflectLine(i, text)
	}
	if e.cursorVisible {
		e.screen.ReflectCursorAt(e.cursorLine, e.cursorChar)
	} else {
		e.screen.ClearCursor()
	}
}

// Stop: Editor 종료
func (e *Editor) Stop() {
	e.running = false
	e.fpsTicker.Stop()
	e.blinkTicker.Stop()
}

// toggleCursorBlink: 커서 깜빡
func (e *Editor) toggleCursorBlink() {
	if e.cursorVisible {
		println("켜짐->꺼짐")
		e.screen.ClearCursor()
		e.cursorVisible = false
	} else {
		e.screen.ReflectCursorAt(e.cursorLine, e.cursorChar)
		println("꺼짐->켜짐")
		e.cursorVisible = true
	}
}
