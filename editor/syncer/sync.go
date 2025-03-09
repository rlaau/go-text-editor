package main

import (
	"go_editor/editor/commander"
	glp "go_editor/editor/screener/glyph"
)

// ----------------------------------------------------
// (1) 기본 구조체 정의
// ----------------------------------------------------

// SyncProtocol은 편집/동기화 관련 처리를 위한 구조체 (여기서는 stub)
type SyncProtocol struct {
	screenWidth  int
	screenHeight int
	LineHeight   int

	fgColor uint32
	bgColor uint32
	cursor  *Cursor

	syncData      *SyncData
	SyncStateCode SyncStateCode
	changedNode   *SyncNode
}

// ----------------------------------------------------
// (3) SyncProtocol 생성자
// ----------------------------------------------------
// TODO 추후 "스크린스펙"받는 로직으로 변경
func NewSyncProtocol(screenWidth, screenHeight int, fg, bg uint32, LineHeight int) *SyncProtocol {
	lineCount := screenHeight / LineHeight
	syncData := &SyncData{}
	// 우선은 라인의 텍스트를 빈 문자열로 다 초기화 해 둚
	//다만 이 단계에선 아직 텍스트-라인 버퍼간 싱크가 없음
	for range lineCount {
		syncData.insertNode(0, "")
	}

	sp := &SyncProtocol{
		screenWidth:   screenWidth,
		screenHeight:  screenHeight,
		LineHeight:    LineHeight,
		fgColor:       fg,
		bgColor:       bg,
		syncData:      syncData,
		SyncStateCode: NodeModified,
		changedNode:   nil,
		cursor:        NewCursor(2, glp.GlyphHeight, 0xFF000000),
	}
	//여기서 워킹 통해서 각 노드마다 싱크 맞춰줌
	syncData.ForEach(func(sn *SyncNode) {
		sp.syncNode(sn)
	})
	return sp
}

// TODO cmd, 커서좌표 받아서 처리 후에
// TODO 커서 좌표 다시 반환함
// TODO 우선적으로 커멘드를 인터프리팅 후에 실행하기
// ex) "\n"들어오면 Slice또는 InsertNode
// ex) Del이면 legth기반으로 Modify or DelNode
func (sp *SyncProtocol) ProcessCommand(cmd commander.Command) (int, int) {
	//TODO 1. 명령 인터프리팅
	//TODO cmd-> 싱크 코드 및 데이터
	//TODO 2. 인터프리트 된 명령어 실행 및 싱크 마킹
	//TODO 이후 1)커서 리턴 2) 싱크 맞추기
	//TODO 이후 3. 커서까지 싱크 맞추기
	// 아직 구체 구현 없음
	return 0, 0
}

func (sp *SyncProtocol) interpretCommand(cmd commander.Command) opSequence {
	//TODO 여기선, 시뮬레이팅 후, "정확한 op코드 생성"을 하기
	//!! 이 함수부터 코딩 시작하기!!!!!!!
	//!! 적당히 인터프리팅 후에
	//!! 모든 초안을 여기서 op시퀀시스에 다 저장!!!
	//!! 이후 프로세스 커멘드에선 해당 op시퀀스에 맞게 작동만 하기!!!
	return nil
}

// TODO ProcessCommand내에서, cmd파싱 후 syncData호출한 후에
// TODO 어느 노드에, 어떤 변경 했는지를 싱크프로토콜에 기록
func (sp *SyncProtocol) markSync() {}

// TODO 싱크마크 바탕으로, 해당 노드 찾아가서 피스테이블에 flatted 호출 후 싱크 맞추기
// TODO 라인버퍼 nil 검증 후 아예 NewLineBuffer를 호출 후에 새로 하기
func (sp *SyncProtocol) resolveSync() {

}

func (sp *SyncProtocol) syncNode(sn *SyncNode) {
	str := sn.PieceTable.String()
	lineBuffer := sn.LineBuffer
	if lineBuffer == nil {
		lineBuffer = sp.NewLineBuffer()
	}
	sp.ReflectLine(lineBuffer, str)
}

// SyncData: 요구사항에서 주어진 구조
// -> 여기에 Insert/Delete/SliceNode/ProcessCommand 메서드를 추가 (이중 연결 리스트 버전)
// TODO 추후 다중초점 혹은 AVL트리 방식으로 노드 관리하게 하기
// TODO 그럼 성능이 더욱 안정적으로 나오게 할 수 있음
// SyncData는 노드를 관리하는 연결 리스트 구조체 (여기서는 이중 연결 리스트)
type SyncData struct {
	head *SyncNode
	// (필요시 tail, length 등 추가 가능)
}

// SyncNode는 두 편집 구조(PieceTable, LineBuffer)를 묶은 노드
type SyncNode struct {
	PieceTable *PieceTable
	LineBuffer *LineBuffer

	prev *SyncNode
	next *SyncNode
}

// 편집 상태 코드 (추후 상세 동기화 시 사용)
type SyncStateCode int

const (
	NodeInserted SyncStateCode = iota
	NodeSliced
	NodeDeleted
	NodeModified
)

type ModifyCode int

const (
	InsertASCII ModifyCode = iota
	DeleteASCII
	HoldASCII
)

func (sd *SyncData) ForEach(fn func(cur *SyncNode)) {
	cur := sd.head
	for cur != nil {
		fn(cur)
		cur = cur.next
	}
}

// ! 항상 프로세싱 커멘드 통해서 호출됨
// ! 여기서 DEL, Insert등은 "정상적인 범위"를 가정한 체로 작동함
// ! 그러므로, 라인 간 구조가 바뀌는 것은 고려할 필요 없음
func (sd *SyncData) modifyNode(n uint, cursorChar int, char rune, modifyCode ModifyCode) {
	cur, found := sd.findNode(n)
	if !found {
		return
	}
	if modifyCode == DeleteASCII {
		cur.PieceTable.DeleteRune(cursorChar)
	}
	if modifyCode == InsertASCII {
		cur.PieceTable.InsertRune(cursorChar, char)
	}

}

// insertNode(n, newData) : n번째 위치에 새 노드를 삽입
//
//	0-based 인덱스에서, 예를 들어, insertNode(0, "hello") → 리스트가 비어있다면 head가 "hello"가 됨.
//	만약 이미 노드가 존재할 경우, insertNode(0, "world")는 새 노드를 head로 만들어 기존 노드가 뒤로 밀림.
func (sd *SyncData) insertNode(n uint, newData string) {
	newNode := &SyncNode{
		PieceTable: NewPieceTable(newData),
		LineBuffer: nil,
		prev:       nil,
		next:       nil,
	}

	// 빈 리스트인 경우
	if sd.head == nil {
		sd.head = newNode
		return
	}

	// 삽입 위치가 0이면, 새 노드를 head로
	if n == 0 {
		newNode.next = sd.head
		sd.head.prev = newNode
		sd.head = newNode
		return
	}

	// 0이 아닌 경우, (n-1)번째 노드를 찾아 그 뒤에 삽입
	cur, found := sd.findNode(n - 1) // (n-1)번째 노드 뒤에 삽입
	if !found {
		return
	}

	// cur가 (n-1)번째 노드임
	newNode.next = cur.next
	newNode.prev = cur
	if cur.next != nil {
		cur.next.prev = newNode
	}
	cur.next = newNode
}

// deleteNode(n): n번째 노드를 삭제 (0-based)
func (sd *SyncData) deleteNode(n uint) {

	cur, found := sd.findNode(n)
	if !found {
		return
	}
	// cur가 n번째 노드
	if cur.prev == nil {
		// head 삭제
		sd.head = cur.next
		if sd.head != nil {
			sd.head.prev = nil
		}
	} else {
		cur.prev.next = cur.next
		if cur.next != nil {
			cur.next.prev = cur.prev
		}
	}
}

// sliceNode(n, char): n번째 노드의 PieceTable.data를 char 인덱스에서 슬라이스
// 슬라이스는 exclusive 연산으로, 앞쪽에는 [0, char) (즉, char 인덱스 미포함), 뒷쪽은 [char, end)
// 예를 들어 "hello"에서 sliceNode(0, 2) → "he", "llo"
// n번째 노드의 data에 대해, n번째 글자가 포함되지 않고, char 인덱스 이전까지만 남김.
func (sd *SyncData) sliceNode(n, char uint) {

	// n번째 노드 찾기
	cur, found := sd.findNode(n) // (n-1)번째 노드 뒤에 삽입
	if !found {
		return
	}

	frontPT, backPT := cur.PieceTable.SlicePieceTable(int(char))

	// n번째 노드는 front로 변경
	cur.PieceTable = frontPT

	// back이 존재하면, 새 노드를 만들어 n번째 노드 뒤에 삽입
	newNode := &SyncNode{
		PieceTable: backPT,
		LineBuffer: nil,
		prev:       cur,
		next:       cur.next,
	}
	if cur.next != nil {
		cur.next.prev = newNode
	}
	cur.next = newNode
}

// findNode(n): 0-based index로 n번째 노드를 찾음
func (sd *SyncData) findNode(n uint) (*SyncNode, bool) {
	if sd.head == nil {
		return nil, false
	}

	cur := sd.head
	count := uint(0)
	for count < n && cur.next != nil {
		cur = cur.next
		count++
	}

	if count != n {
		return nil, false // n이 리스트 길이보다 크면 false 반환
	}

	return cur, true
}
