package editor

import (
	"fmt"

	"go_editor/editor/commander"
	"go_editor/editor/screener"
	"time"

	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/keybind"
)

// Editor: screener를 가지고, FPS 기반 화면 업데이트 + 커서 깜빡임 + 이벤트 처리
type Editor struct {
	screener    *screener.Screener
	commander   *commander.Commander
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
	xu            *xgbutil.XUtil
}

// NewEditor: Editor 인스턴스 생성
func NewEditor(width, height int, fps int) (*Editor, error) {
	xu, err := xgbutil.NewConn()
	if err != nil {
		return nil, fmt.Errorf("XGBUtil 연결 실패: %v", err)
	}

	scr, err := screener.NewScreener(xu, width, height)
	if err != nil {
		return nil, err
	}

	// Commandor 생성
	cmdor := commander.NewCommandor(xu)
	e := &Editor{
		screener:      scr,
		commander:     cmdor, // Commandor 위임
		xu:            xu,
		fpsTicker:     time.NewTicker(time.Second / time.Duration(fps)), // 30FPS
		blinkTicker:   time.NewTicker(time.Second * 1),                  // 1초 주기
		running:       true,
		lines:         []string{"Hello", "KeyPress Count: 0"}, // 초기 2개 라인,
		textCount:     0,
		cursorVisible: false,
		// 커서는 line=1, char=3 초기값
		cursorLine: 0,
		cursorChar: 0,
	}
	// X 키 바인딩 초기화
	keybind.Initialize(xu)
	return e, nil
}

// Run: 메인 이벤트 루프
func (e *Editor) Run() {
	// 이벤트 전용 고루틴: 블로킹 WaitForEvent() → eventChan 로 전달
	e.reflectAll()
	e.commander.StartListening()

	for e.running {
		select {
		case <-e.blinkTicker.C:
			// 1초마다 커서 깜빡
			e.toggleCursorBlink()

		case <-e.fpsTicker.C:
			// 30FPS로 화면 Flush
			e.screener.FlushBuffer()

		case cmd, ok := <-e.commander.GetCommandChan():
			if !ok {
				e.running = false
				break
			}
			e.processCommand(cmd)
		}
	}
}

// processCommand: Command를 처리
func (e *Editor) processCommand(cmd commander.Command) {
	switch cmd.Code {
	case commander.CmdExit:
		println("엑싯")
		e.running = false
	case commander.CmdDelete:
		println("딜릿")
	case commander.CmdInsert:
		println("인서트")
	case commander.CmdMove:
		println("무브")
		if charInput, ok := cmd.Input.(commander.CharInput); ok {
			switch charInput.Char {
			case commander.KeyUp:
				e.cursorLine = max(0, e.cursorLine-1)
			case commander.KeyDown:
				e.cursorLine++ // 줄 수 제한 없음, 필요하면 maxLine 체크 추가
			case commander.KeyLeft:
				e.cursorChar = max(0, e.cursorChar-1)
			case commander.KeyRight:
				e.cursorChar++ // 글자 수 제한 없음, 필요하면 줄 길이 체크 추가
			}
			e.cursorVisible = true

		}

	}

	e.lines[1] = fmt.Sprintf("KeyPress Count: %d", e.textCount)
	e.textCount++
	e.reflectAll()
}

// min/max 유틸 함수 정의 (Go 1.20 이상이면 math 패키지 사용 가능)
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// reflectAll: 모든 라인을 스크리너에 반영
func (e *Editor) reflectAll() {
	e.screener.Clear(0xFFFFFFFF)

	// 간단: line 0 -> y=50, line1 -> y= 70
	for i, text := range e.lines {
		e.screener.ReflectLine(i, text)
	}
	if e.cursorVisible {
		e.screener.ReflectCursorAt(e.cursorLine, e.cursorChar)
	} else {
		e.screener.ClearCursor()
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
		e.screener.ClearCursor()
		e.cursorVisible = false
	} else {
		e.screener.ReflectCursorAt(e.cursorLine, e.cursorChar)
		e.cursorVisible = true
	}
}
