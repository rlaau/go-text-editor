package main

import (
	"fmt"
)

// -------------------------------------------------------------
// 1) 연결 리스트 예시: PieceTable 측과 LineBuffer 측
// -------------------------------------------------------------

// LinkedListOfPieceTable: PieceTable 기반의 연결 리스트(구조만 스케치)
type LinkedListOfPieceTable struct {
	head *PieceTableNode
	// 필요에 따라 추가 필드
}

// PieceTableNode: 실제 노드 구조 (구체 필드는 필요에 따라 확장)
type PieceTableNode struct {
	ID        int    // 노드 식별자
	PieceData string // 예시
	next      *PieceTableNode
}

// Flatten: 이 연결 리스트를 직렬화(Flatten)하는 예시 함수
func (l *LinkedListOfPieceTable) Flatten() string {
	// 간단히 노드별 데이터를 합치는 정도로 가정
	// 실제 구현에서는 구조에 맞게 JSON, XML, 바이너리 직렬화 등 가능
	var result string
	for cur := l.head; cur != nil; cur = cur.next {
		result += cur.PieceData + "|"
	}
	return result
}

// InsertNode, DeleteNode 등등 필요한 메서드는 실제 구현에서 추가
func (l *LinkedListOfPieceTable) InsertNode(nodeNumber int) {
	// 예시 로직
	fmt.Printf("[PieceTable] 노드 삽입: nodeNumber=%d\n", nodeNumber)
	// ...
}

// ReflectFlattened: 다른 쪽에서 받은 직렬화 데이터를 반영(역직렬화)하는 예시 함수
func (l *LinkedListOfPieceTable) ReflectFlattened(data string) {
	fmt.Printf("[PieceTable] 직렬화된 데이터 반영: %s\n", data)
	// ...
}

// -------------------------------------------------------------

// LinkedListOfLineBuffer: LineBuffer 기반의 연결 리스트
type LinkedListOfLineBuffer struct {
	head *LineBufferNode
}

// LineBufferNode: 실제 노드 구조 (구체 필드는 필요에 따라 확장)
type LineBufferNode struct {
	ID         int
	LineBuffer string
	next       *LineBufferNode
}

// Flatten
func (l *LinkedListOfLineBuffer) Flatten() string {
	var result string
	for cur := l.head; cur != nil; cur = cur.next {
		result += cur.LineBuffer + "\n"
	}
	return result
}

func (l *LinkedListOfLineBuffer) ReflectFlattened(data string) {
	fmt.Printf("[LineBuffer] 직렬화된 데이터 반영: %s\n", data)
	// ...
}

// InsertNode, DeleteNode 등등 필요한 메서드는 실제 구현에서 추가
func (l *LinkedListOfLineBuffer) InsertNode(nodeNumber int) {
	fmt.Printf("[LineBuffer] 노드 삽입: nodeNumber=%d\n", nodeNumber)
	// ...
}

// -------------------------------------------------------------
// 2) 프로토콜 관련
// -------------------------------------------------------------

// LinkedListStateCode: 두 연결 리스트(노드) 간 동기화 이벤트 타입
type LinkedListStateCode int

const (
	NodeInserted LinkedListStateCode = iota
	NodeSliced
	NodeDeleted
	Hold
)

// ProtocolCommand: 명령 종류(간단 예시)
type ProtocolCommand string

const (
	CommandInsert ProtocolCommand = "Insert"
	CommandDelete ProtocolCommand = "Delete"
	CommandModify ProtocolCommand = "Modify"
	// ...
)

// SyncProtocol: 두 연결 리스트를 중재·동기화하는 "프로토콜" 구조체
type SyncProtocol struct {
	// 두 개의 데이터 구조체: PieceTable 기반, LineBuffer 기반
	pieceTableList *LinkedListOfPieceTable
	lineBufferList *LinkedListOfLineBuffer

	// 노드 동기화 상태 코드
	linkedListStateCode LinkedListStateCode

	// 동기화할 노드 ID
	syncedNodeId int
}

// NewSyncProtocol: 생성자 예시
func NewSyncProtocol(ptList *LinkedListOfPieceTable, lbList *LinkedListOfLineBuffer) *SyncProtocol {
	return &SyncProtocol{
		pieceTableList:      ptList,
		lineBufferList:      lbList,
		linkedListStateCode: Hold, // 초기 상태
		syncedNodeId:        -1,   // 초기: 없음
	}
}

// ProcessCommand: 프로토콜의 핵심 메서드
// nodeNumber와 charNumber, command를 인자로 받아서
// 내부적으로 "시뮬레이션" 후, 두 리스트의 노드 개수·순서를 동기화
func (sp *SyncProtocol) ProcessCommand(nodeNumber, charNumber int, command ProtocolCommand) {
	// 간단한 예시 로직(실제 구현 시 필요한 작업)
	fmt.Printf("[Protocol] 명령 처리 시작: nodeNumber=%d, charNumber=%d, command=%s\n",
		nodeNumber, charNumber, command)

	// 1) 우선 PieceTable 쪽에서 명령 시뮬레이션
	simulateResult := sp.simulateOnPieceTable(nodeNumber, charNumber, command)

	// 2) 시뮬레이션 결과(노드 삽입/삭제/분할/유지)에 따라 linkedListStateCode, syncedNodeId 결정
	sp.linkedListStateCode = simulateResult.stateCode
	sp.syncedNodeId = nodeNumber // 일단 예시로 nodeNumber를 그대로 설정

	// 3) 필요한 경우, 두 리스트를 동기화
	sp.syncLinkedLists()

	// 4) 최종적으로, Flatten → ReflectFlattened 과정도 수행할 수 있음 (상황에 따라)
	//    예: PieceTable -> Flatten -> LineBuffer -> ReflectFlattened
	flattedFromPiece := sp.pieceTableList.Flatten()
	sp.lineBufferList.ReflectFlattened(flattedFromPiece)
}

// simulateOnPieceTable: 간단히 "이 명령이 PieceTable에 어떤 영향을 줄까?" 를 시뮬레이션
func (sp *SyncProtocol) simulateOnPieceTable(nodeNumber, charNumber int, command ProtocolCommand) (result SimulationResult) {
	// 예시: node 삽입 명령이라면
	switch command {
	case CommandInsert:
		// (가정) 노드 삽입이 필요하다면
		result.stateCode = NodeInserted
	case CommandDelete:
		result.stateCode = NodeDeleted
	default:
		result.stateCode = Hold
	}
	return
}

// syncLinkedLists: 시뮬레이션 결과를 토대로, PieceTable과 LineBuffer 쪽 노드 개수/순서를 맞춰주는 로직
func (sp *SyncProtocol) syncLinkedLists() {
	fmt.Printf("[Protocol] syncLinkedLists: stateCode=%v, syncedNodeId=%d\n",
		sp.linkedListStateCode, sp.syncedNodeId)

	switch sp.linkedListStateCode {
	case NodeInserted:
		// 예시: 두 리스트 모두 해당 nodeNumber 위치에 노드를 삽입
		sp.pieceTableList.InsertNode(sp.syncedNodeId)
		sp.lineBufferList.InsertNode(sp.syncedNodeId)
	case NodeDeleted:
		// 예시: 두 리스트 모두 해당 nodeNumber 위치 노드를 삭제
		// (DeleteNode는 여기선 예시, 실제 메서드가 필요)
		// sp.pieceTableList.DeleteNode(sp.syncedNodeId)
		// sp.lineBufferList.DeleteNode(sp.syncedNodeId)
		fmt.Println(">> 노드 삭제 로직 실행(예시)")
	case NodeSliced:
		fmt.Println(">> 노드 분할 로직 실행(예시)")
	case Hold:
		fmt.Println(">> 노드 변경 없음")
	}
}

// SimulationResult: 시뮬레이션 결과(예시)
type SimulationResult struct {
	stateCode LinkedListStateCode
}

func main() {
	// 예시로 PieceTable과 LineBuffer 리스트 생성
	ptList := &LinkedListOfPieceTable{}
	lbList := &LinkedListOfLineBuffer{}

	// 프로토콜 생성
	syncProto := NewSyncProtocol(ptList, lbList)

	// 명령 처리해보기
	syncProto.ProcessCommand(2, 10, CommandInsert)
	syncProto.ProcessCommand(3, 0, CommandDelete)
	syncProto.ProcessCommand(4, 5, CommandModify)
}
