package main

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
type SyncData struct {
	head *SyncNode
	// 필요하면 tail, length 등을 둘 수도 있음
}

// SyncNode: 이중 연결 리스트를 구성하기 위해 prev, next 모두 추가
type SyncNode struct {
	PieceTable PieceTable
	LineBuffer LineBuffer

	prev *SyncNode
	next *SyncNode
}

type PieceTable struct {
	data string
}
type LineBuffer struct {
	data []uint32
}

type SyncStateCode int

const (
	NodeInserted SyncStateCode = iota
	NodeSliced
	NodeDeleted
	Hold
)

// ----------------------------------------------------
// (1) insert(n, newData string)
//
//	n번째 노드 "뒤"에 새 노드를 삽입 (0-based index)
//

func (sd *SyncData) insert(n uint, newData string) {

	newNode := &SyncNode{
		PieceTable: PieceTable{data: newData},
		LineBuffer: LineBuffer{},
		prev:       nil,
		next:       nil,
	}

	if sd.head == nil {
		// 리스트가 비어있다면 새 노드를 head로 설정
		sd.head = newNode
		return
	}

	// n번째 노드 찾아 이동
	cur := sd.head
	count := 0
	for count < int(n) && cur.next != nil {
		cur = cur.next
		count++
	}
	// 현재 cur가 "n번째 노드"
	// => 그 "뒤"에 newNode를 삽입

	newNode.prev = cur
	newNode.next = cur.next
	if cur.next != nil {
		cur.next.prev = newNode
	}
	cur.next = newNode
}

// ----------------------------------------------------
// (2) delete(n int)
//
//	n번째 노드를 삭제 (0-based index)
//
// ----------------------------------------------------
func (sd *SyncData) delete(n uint) {
	if sd.head == nil {
		return
	}

	cur := sd.head
	count := uint(0)

	for count < n && cur.next != nil {
		cur = cur.next
		count++
	}
	// 현재 cur가 n번째 노드
	if count != n {
		// n이 리스트 길이보다 큼 -> 삭제 불가
		return
	}

	// 이제 cur를 제거
	if cur.prev == nil {
		// head 노드 삭제
		sd.head = cur.next
		if sd.head != nil {
			sd.head.prev = nil
		}
	} else {
		// 중간/마지막 노드 삭제
		cur.prev.next = cur.next
		if cur.next != nil {
			cur.next.prev = cur.prev
		}
	}
}

// ----------------------------------------------------
// (3) sliceNode(n, char int)
//
//	n번째 노드의 PieceTable.data를 char 지점에서 슬라이스
//	뒷부분은 새 노드로 만들어 n번째 노드 "뒤"에 삽입
//
// 코드 보면 알겠지만, exclusive한 연산임
// 2라하면, 0,1포함, 2는 배제하는 슬라이싱
// 직관적으론, "n번쨰 글자가지 포함시키고, 그 이후 배제"로 보면 될듯
// "hello"를 5기준 슬라이스 하면 "hello",""로 됨
// ----------------------------------------------------
func (sd *SyncData) sliceNode(n, char uint) {
	if sd.head == nil {
		return
	}

	// n번째 노드 찾기
	cur := sd.head
	count := uint(0)
	for count < n && cur.next != nil {
		cur = cur.next
		count++
	}
	if count != n {
		// 범위 벗어남
		return
	}

	// 이제 cur가 n번째 노드
	original := cur.PieceTable.data
	if int(char) > len(original) {
		return
	}

	front := original[:char]
	back := original[char:]

	// (a) n번째 노드에는 front만 남긴다
	cur.PieceTable.data = front

	// (b) 뒷부분(back)은 새 노드로 만들어 n번째 노드 뒤에 삽입
	newNode := &SyncNode{
		PieceTable: PieceTable{data: back},
		LineBuffer: LineBuffer{},
		prev:       cur,
		next:       cur.next,
	}
	// 링크 연결
	if cur.next != nil {
		cur.next.prev = newNode
	}
	cur.next = newNode
}

// ----------------------------------------------------
// SyncProtocol에 대한 생성자 (간단 예시)
// ----------------------------------------------------
func NewSyncProtocol() *SyncProtocol {
	return &SyncProtocol{
		syncData:      &SyncData{},
		SyncStateCode: Hold,
		changedNode:   nil,
	}
}
