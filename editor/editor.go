package editor

import (
	"fmt"
	"unicode"

	"go_editor/editor/screener"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/keybind"
)

//TODO 텍스트의 관리를 Screener에서 뺏어오기
//TODO 스크리너는 걍 텍스트 "받아서" 그 인자 바탕으로 렌더링만 구현
//TODO 그 텍스트를 주고, 또 메니징 하는 것은 에디터가 하기. rope구조 등으로

// TODO 이후, "명령어"는 커멘더에 위임해서 처리해보기. xgb의 이벤트를 래핑해서 인터프리트
type CommandCode uint8

const (
	CmdMove CommandCode = iota
	CmdAppend
	CmdInsert
	CmdDelete
	CmdExit
)

// 모든 입력 타입이 구현해야 하는 인터페이스
type CommandInput interface {
	IsCommandInput() // 마커 인터페이스 역할
}

// `rune`을 감싸는 구조체
type CharInput struct {
	Char rune
}

// `CharInput`이 `CommandInput`을 구현
func (c CharInput) IsCommandInput() {}

// 커서 이동을 위한 구조체
type ClickInput struct {
	Height, Width int
}

// `CursorPosition`도 `CommandInput`을 구현
func (c ClickInput) IsCommandInput() {}

// Command: 에디터가 처리할 명령 정보
type Command struct {
	Code  CommandCode  // 명령 코드
	Input CommandInput // 입력 값 (현재는 키보드 `rune` 만 사용)
}

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
	xu            *xgbutil.XUtil
	eventChan     chan Command
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

	if err != nil {
		return nil, err
	}

	e := &Editor{
		screen:        scr,
		xu:            xu,
		fpsTicker:     time.NewTicker(time.Second / time.Duration(fps)), // 30FPS
		blinkTicker:   time.NewTicker(time.Second * 1),                  // 1초 주기
		running:       true,
		lines:         []string{"Hello", "KeyPress Count: 0"}, // 초기 2개 라인,
		textCount:     0,
		cursorVisible: false,
		// 커서는 line=1, char=3 초기값
		cursorLine: 1,
		cursorChar: 3,
		eventChan:  make(chan Command, 20), // Command 채널
	}
	// X 키 바인딩 초기화
	keybind.Initialize(xu)
	return e, nil
}

// TranslateXEventToCommand: X 이벤트를 Command로 변환
func TranslateXEventToCommand(xu *xgbutil.XUtil, ev xgb.Event) (Command, bool) {
	switch e := ev.(type) {
	case xproto.KeyPressEvent:
		keyRune, err := TranslateKeyCode(xu, e.Detail, e.State)

		if err != nil {
			return Command{}, false
		}

		var cmd CommandCode
		switch keyRune {
		case KeyESC:
			cmd = CmdExit
		case KeyBackSpace:
			cmd = CmdDelete
		case KeyLeft, KeyRight, KeyDown, KeyUp:
			cmd = CmdMove
		case KeyEnter1, KeyEnter2:
			cmd = CmdInsert
		default:
			cmd = CmdInsert
		}
		println("받은 룬", keyRune)

		return Command{
			Code:  cmd,
			Input: CharInput{keyRune},
		}, true

	default:
		return Command{}, false
	}
}

// TranslateKeyCode: KeyCode를 KeySym으로 변환 후 rune으로 변환

// X11 KeySym 상수 정의 (X11/keysymdef.h 참고)
const (
	XK_ESC       = 0xFF1B
	XK_Return    = 0xFF0D
	XK_KP_Enter  = 0xFF8D
	XK_BackSpace = 0xFF08
	XK_Left      = 0xFF51
	XK_Right     = 0xFF53
	XK_Up        = 0xFF52
	XK_Down      = 0xFF54
)

const (
	KeyESC       rune = 0xFF1B
	KeyBackSpace rune = 0xFF08
	KeyLeft      rune = 0xFF51
	KeyRight     rune = 0xFF53
	KeyUp        rune = 0xFF52
	KeyDown      rune = 0xFF54
	KeyEnter1    rune = '\n'
	KeyEnter2    rune = 0xFF0D
)

// TranslateKeyCode: KeyCode를 KeySym으로 변환 후 rune으로 변환

// TranslateKeyCode: KeyCode를 KeySym으로 변환 후 rune으로 변환
func TranslateKeyCode(xu *xgbutil.XUtil, keycode xproto.Keycode, state uint16) (rune, error) {
	// keycode를 KeySym으로 변환
	keysym := keybind.KeysymGet(xu, keycode, 0)
	if keysym == 0 {
		return 0, fmt.Errorf("no keysym found for keycode %d", keycode)
	}

	// 특수키 매핑
	switch keysym {
	case XK_ESC:
		return KeyESC, nil
	case XK_Return, XK_KP_Enter:
		return KeyEnter1, nil
	case XK_BackSpace:
		return KeyBackSpace, nil
	case XK_Left:
		return KeyLeft, nil
	case XK_Right:
		return KeyRight, nil
	case XK_Up:
		return KeyUp, nil
	case XK_Down:
		return KeyDown, nil
	}

	// 일반 문자 키 처리
	keysymStr := keybind.LookupString(xu, state, keycode)
	if keysymStr == "" {
		return 0, fmt.Errorf("failed to convert keysym to string")
	}

	runes := []rune(keysymStr)
	if len(runes) == 0 {
		return 0, fmt.Errorf("failed to convert keysym to rune")
	}

	// Shift 키 처리
	if state&xproto.ModMaskShift > 0 {
		return unicode.ToUpper(runes[0]), nil
	}
	return runes[0], nil
}

// collectEvents: X 이벤트를 수신하여 Command 변환 후 eventChan으로 전달
func (e *Editor) collectEvents() {
	for {
		ev, err := e.xu.Conn().WaitForEvent()
		if err != nil {
			close(e.eventChan)
			return
		}
		if ev != nil {
			cmd, ok := TranslateXEventToCommand(e.xu, ev)
			println("코드", cmd.Code, "값", cmd.Input)
			if ok {
				e.eventChan <- cmd
			}
		}
	}
}

// Run: 메인 이벤트 루프
func (e *Editor) Run() {
	// 이벤트 전용 고루틴: 블로킹 WaitForEvent() → eventChan 로 전달
	e.redrawAll()
	go e.collectEvents()

	for e.running {
		select {
		case <-e.blinkTicker.C:
			// 1초마다 커서 깜빡
			e.toggleCursorBlink()

		case <-e.fpsTicker.C:
			// 30FPS로 화면 Flush
			e.screen.FlushBuffer()

		case cmd, ok := <-e.eventChan:
			if !ok {
				e.running = false
				break
			}
			e.handleCommand(cmd)
		}
	}
}

// handleCommand: Command를 처리
func (e *Editor) handleCommand(cmd Command) {
	switch cmd.Code {
	case CmdExit:
		println("엑싯")
		e.running = false
	case CmdDelete:
		println("딜릿")
	case CmdInsert:
		println("인서트")
	case CmdMove:
		println("무브")
	}

	e.lines[1] = fmt.Sprintf("KeyPress Count: %d", e.textCount)
	e.textCount++
	e.redrawAll()
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
		e.screen.ClearCursor()
		e.cursorVisible = false
	} else {
		e.screen.ReflectCursorAt(e.cursorLine, e.cursorChar)
		e.cursorVisible = true
	}
}
