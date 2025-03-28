package editor

import (
	"fmt"
	"log"
	"os"

	"go_editor/editor/commander"
	"go_editor/editor/handlefile"
	"go_editor/editor/screener"
	"go_editor/editor/syncer"
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
	xu            *xgbutil.XUtil

	syncProtocol *syncer.SyncProtocol
}

// NewEditor: Editor 인스턴스 생성
func NewEditor(width, height int, fps int) (*Editor, error) {
	xu, err := xgbutil.NewConn()
	if err != nil {
		return nil, fmt.Errorf("XGBUtil 연결 실패: %v", err)
	}
	savePath := handlefile.GetSaveTxtPath()
	var syncProtocol *syncer.SyncProtocol

	// 파일 존재 여부 및 내용 확인
	fileInfo, err := os.Stat(savePath)
	if err != nil || fileInfo.Size() == 0 {
		// 파일이 없거나 비어있으면 NewSyncProtocol 호출
		if os.IsNotExist(err) {
			log.Printf("🆕 파일이 존재하지 않아 새 문서를 생성합니다: %s", savePath)
		} else if err == nil && fileInfo.Size() == 0 {
			log.Printf("🆕 파일이 비어있어 새 문서를 생성합니다: %s", savePath)
		} else {
			log.Printf("⚠️ 파일 접근 오류: %v, 새 문서를 생성합니다", err)
		}
		syncProtocol = syncer.NewSyncProtocol(width, height, 0xFF000000, 0xFFFFFFFF, 16)
	} else {
		// 파일이 존재하고 내용이 있으면 LoadSyncProtocol 호출
		log.Printf("📄 기존 파일을 불러옵니다: %s (크기: %d 바이트)", savePath, fileInfo.Size())
		syncProtocol = syncer.LoadSyncProtocol(width, height, 0xFF000000, 0xFFFFFFFF, 16)
	}
	scr, err := screener.NewScreener(xu, width, height, 0xFF000000, 0xFFFFFFFF)
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

		syncProtocol: syncProtocol,
	}
	// X 키 바인딩 초기화
	keybind.Initialize(xu)
	return e, nil
}

// Run: 메인 이벤트 루프
func (e *Editor) Run() {
	defer e.syncProtocol.SaveToFile()

	e.commander.StartListening()

	for e.running {
		select {
		case <-e.blinkTicker.C:
			// 1초마다 커서 깜빡
			e.toggleCursorBlink()

		case <-e.fpsTicker.C:
			// 30FPS로 화면 Flush

			e.screener.FlushBuffer(e.syncProtocol.FlushLineBuffer())

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
	//레이어 2 수정
	e.syncProtocol.ClearCursor()
	//레이어 1 수정
	isContinue := e.syncProtocol.ProcessCommand(cmd)
	if !isContinue {
		e.running = false
		return
	}
	//레이어 2 수정
	if e.syncProtocol.IsCursorVisible() {
		e.syncProtocol.CursorDrawOn()
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
	if e.syncProtocol.IsCursorVisible() {
		e.syncProtocol.ClearCursor()
	} else {
		e.syncProtocol.CursorDrawOn()
	}
}
