package main

// ----------------------------------------------------
// (1) 기본 구조체 정의
// ----------------------------------------------------

// SyncProtocol은 편집/동기화 관련 처리를 위한 구조체 (여기서는 stub)
type SyncProtocol struct {
	syncData      *SyncData
	SyncStateCode SyncStateCode
	changedNode   *SyncNode
}

// TODO cmd, 커서좌표 받아서 처리 후에
// TODO 커서 좌표 다시 반환함
// 한번에 하나씩 처리한다는 것을 잊지 말기
// 하나 처리 후, 그떄마다 싱크 맞추기
func (sp *SyncProtocol) ProcessCommand(cmd string, cursorX int, cursorY int) (int, int) {
	// 아직 구체 구현 없음
	return 0, 0
}

// TODO ProcessCommand내에서, cmd파싱 후 syncData호출한 후에
// TODO 어느 노드에, 어떤 변경 했는지를 싱크프로토콜에 기록
func (sp *SyncProtocol) markSync() {}

// TODO 싱크마크 바탕으로, 해당 노드 찾아가서 피스테이블에 flatted 호출 후 싱크 맞추기
func (sp *SyncProtocol) resolveSync() {

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

// 피스테이블 데이터를 라인버퍼에 동기화
func (sn *SyncNode) syncData() {}

type LineBuffer struct {
	data []uint32
}

// 편집 상태 코드 (추후 상세 동기화 시 사용)
type SyncStateCode int

const (
	NodeInserted SyncStateCode = iota
	NodeSliced
	NodeDeleted
	Hold
)

// ----------------------------------------------------
// (2) SyncData 메서드 : insert, delete, sliceNode (모두 zero-based)
// ----------------------------------------------------

// insert(n, newData) : n번째 위치에 새 노드를 삽입
//
//	0-based 인덱스에서, 예를 들어, insert(0, "hello") → 리스트가 비어있다면 head가 "hello"가 됨.
//	만약 이미 노드가 존재할 경우, insert(0, "world")는 새 노드를 head로 만들어 기존 노드가 뒤로 밀림.
func (sd *SyncData) insert(n uint, newData string) {
	newNode := &SyncNode{
		PieceTable: NewPieceTable(newData),
		LineBuffer: &LineBuffer{},
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
	cur := sd.head
	count := uint(0)
	for count < n-1 && cur.next != nil {
		cur = cur.next
		count++
	}
	// cur가 (n-1)번째 노드임
	newNode.next = cur.next
	newNode.prev = cur
	if cur.next != nil {
		cur.next.prev = newNode
	}
	cur.next = newNode
}

// delete(n): n번째 노드를 삭제 (0-based)
func (sd *SyncData) delete(n uint) {
	if sd.head == nil {
		return
	}
	cur := sd.head
	var count uint = 0
	for count < n && cur.next != nil {
		cur = cur.next
		count++
	}
	if count != n {
		// n이 리스트 길이보다 큰 경우 아무 작업도 하지 않음
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
	if sd.head == nil {
		return
	}
	// n번째 노드 찾기
	cur := sd.head
	var count uint = 0
	for count < n && cur.next != nil {
		cur = cur.next
		count++
	}
	if count != n {
		return // 범위 초과
	}

	frontPT, backPT := cur.PieceTable.SlicePieceTable(int(char))

	// n번째 노드는 front로 변경
	cur.PieceTable = frontPT

	// back이 존재하면, 새 노드를 만들어 n번째 노드 뒤에 삽입
	newNode := &SyncNode{
		PieceTable: backPT,
		LineBuffer: &LineBuffer{},
		prev:       cur,
		next:       cur.next,
	}
	if cur.next != nil {
		cur.next.prev = newNode
	}
	cur.next = newNode
}

// ----------------------------------------------------
// (3) SyncProtocol 생성자
// ----------------------------------------------------
func NewSyncProtocol() *SyncProtocol {
	return &SyncProtocol{
		syncData:      &SyncData{},
		SyncStateCode: Hold,
		changedNode:   nil,
	}
}
