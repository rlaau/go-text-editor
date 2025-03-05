package commander

import (
	"fmt"
	"unicode"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/keybind"
)

// CommandCode: 명령 코드
type CommandCode uint8

const (
	CmdMove CommandCode = iota
	CmdAppend
	CmdInsert
	CmdDelete
	CmdExit
)

// X11 KeySym 상수 정의 (X11/keysymdef.h 참고)
const (
	XK_ESC       = 0xFF1B
	XK_Return    = 0xFF0D
	XK_KP_Enter1 = 0xFF8D
	XK_KP_Enter2 = '\n'
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

// CommandInput 인터페이스
type CommandInput interface {
	IsCommandInput()
}

// CharInput: `rune`을 감싸는 구조체
type CharInput struct {
	Char rune
}

func (c CharInput) IsCommandInput() {}

// ClickInput: 마우스 클릭 입력
type ClickInput struct {
	Height, Width int
}

func (c ClickInput) IsCommandInput() {}

// Command: 실행할 명령
type Command struct {
	Code  CommandCode
	Input CommandInput
}

// Commander: 이벤트 수집 및 처리 담당
type Commander struct {
	xu        *xgbutil.XUtil
	eventChan chan Command
}

func NewCommandor(xu *xgbutil.XUtil) *Commander {
	return &Commander{
		xu:        xu,
		eventChan: make(chan Command, 20),
	}
}

// TranslateXEventToCommand: X 이벤트 -> Command 변환
func (c *Commander) TranslateXEventToCommand(ev xgb.Event) (Command, bool) {
	switch e := ev.(type) {
	case xproto.KeyPressEvent:
		keyRune, err := TranslateKeyCode(c.xu, e.Detail, e.State)
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

		return Command{
			Code:  cmd,
			Input: CharInput{keyRune},
		}, true
	default:
		return Command{}, false
	}
}

// collectCommands: X 이벤트를 수신하고 Command로 변환
func (c *Commander) collectCommands() {
	for {
		ev, err := c.xu.Conn().WaitForEvent()
		if err != nil {
			close(c.eventChan)
			return
		}
		if ev != nil {
			cmd, ok := c.TranslateXEventToCommand(ev)
			if ok {
				c.eventChan <- cmd
			}
		}
	}
}

// StartListening: 이벤트 루프 실행 (별도 고루틴)
func (c *Commander) StartListening() {
	go c.collectCommands()
}

// GetCommandChan: Command 채널 반환
func (c *Commander) GetCommandChan() chan Command {
	return c.eventChan
}

// TranslateKeyCode: KeyCode를 KeySym으로 변환 후 rune으로 변환
func TranslateKeyCode(xu *xgbutil.XUtil, keycode xproto.Keycode, state uint16) (rune, error) {
	// keycode를 KeySym으로 변환
	keysym := keybind.KeysymGet(xu, keycode, 0)
	if keysym == 0 {
		return 0, fmt.Errorf("no keysym found for keycode %d", keycode)
	}

	// 특수키 매핑
	// 심볼 단계에서 얼리 리턴해야 함. 그래야 이후에 string 룩업 가능
	switch keysym {
	case XK_ESC:
		return KeyESC, nil
	case XK_Return, XK_KP_Enter1, XK_KP_Enter2:
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
