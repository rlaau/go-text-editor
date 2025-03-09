package main

// -----------------------------------
// opSequences 구조체
//   - head: 연결 리스트의 시작 노드
//   - tail: 연결 리스트의 마지막 노드
//
// -----------------------------------
type opSequences struct {
	head opSequence
	tail opSequence
}

func NewOpSequences() *opSequences {
	return &opSequences{}
}

// Append() : 새로운 opSequence(노드 체인)를 연결 리스트 끝에 붙인다.
func (ops *opSequences) Append(seq opSequence) {
	if seq == nil {
		return
	}

	// 1) 연결 리스트가 비어있다면 (head == nil)
	//    head와 tail을 seq로 설정
	if ops.head == nil {
		ops.head = seq
		ops.tail = getTail(seq)
		return
	}

	// 2) 비어있지 않다면 tail에 seq를 연결
	//   - tail.nextOp = seq
	//   - 그 다음, tail을 새로 추가된 체인의 마지막 노드로 갱신
	linkTail(ops.tail, seq)
	ops.tail = getTail(seq)
}

// ForEach() : 모든 노드를 순회하며, 각 노드의 opInfo를 fn으로 전달
func (ops *opSequences) ForEach(fn func(info opInfo)) {
	current := ops.head
	for current != nil {
		fn(current.op())
		current = current.next()
	}
}

// getTail() : 전달받은 시퀀스(seq)의 마지막 노드를 찾아 반환
func getTail(start opSequence) opSequence {
	cur := start
	for cur != nil {
		if cur.next() == nil {
			return cur
		}
		cur = cur.next()
	}
	return nil
}

// linkTail() : tail 노드에 seq 노드를 연결
func linkTail(tail, seq opSequence) {
	switch t := tail.(type) {
	case *OpNodeGroup:
		t.nextOp = seq
	case *OpNodeTextOp:
		t.nextOp = seq
	case *OpNodeSyncOp:
		t.nextOp = seq
	case *OpNodeCursorOp:
		t.nextOp = seq
	}
}

// -----------------------------------
// 아래부터는 기존 코드 (인터페이스, 노드들) 그대로 사용
// -----------------------------------

type opSequence interface {
	op() opInfo
	next() opSequence
}

type opKind int

const (
	OpKindNodeGroup opKind = iota
	OpKindNodeText
	OpKindSync
	OpKindCursor
)

type opInfo struct {
	opKind     opKind
	opCode     int
	targetNode *SyncNode
	char       rune
}

// -----------------------------------
// NodeGroup 관련 상수 & 노드
// -----------------------------------
type NodeGroupOp int

const (
	OpInsertNode NodeGroupOp = iota
	OpDeleteNode
	OpModifyNode
	OpHoldNode
)

// OpNodeGroup: opSequence 구현체 (NodeGroup 연산)
type OpNodeGroup struct {
	opCode     NodeGroupOp
	targetNode *SyncNode
	nextOp     opSequence
}

func (ng *OpNodeGroup) op() opInfo {
	return opInfo{
		opKind:     OpKindNodeGroup,
		opCode:     int(ng.opCode),
		targetNode: ng.targetNode,
		char:       ' ', // NodeGroup에서는 char 사용 안 함
	}
}
func (ng *OpNodeGroup) next() opSequence {
	return ng.nextOp
}

// -----------------------------------
// NodeText 관련 상수 & 노드
// -----------------------------------
type NodeTextOp int

const (
	OpInsertRune NodeTextOp = iota
	OpDeleteRune
	OpHoldRune
)

// OpNodeTextOp: opSequence 구현체 (NodeText 연산)
type OpNodeTextOp struct {
	opCode NodeTextOp
	char   rune
	nextOp opSequence
}

func (nt *OpNodeTextOp) op() opInfo {
	return opInfo{
		opKind:     OpKindNodeText,
		opCode:     int(nt.opCode),
		targetNode: nil, // 텍스트 관련이라 targetNode는 사용 안 함
		char:       nt.char,
	}
}
func (nt *OpNodeTextOp) next() opSequence {
	return nt.nextOp
}

// -----------------------------------
// SyncOp 관련 상수 & 노드
// -----------------------------------
type SyncOp int

const (
	OpInsertedSync SyncOp = iota
	OpSlicedSync
	OpModifiedSync
	OpDeletedSync
	OpHoldSync
)

// OpNodeSyncOp: opSequence 구현체 (Sync 연산)
type OpNodeSyncOp struct {
	opCode     SyncOp
	targetNode *SyncNode
	nextOp     opSequence
}

func (so *OpNodeSyncOp) op() opInfo {
	return opInfo{
		opKind:     OpKindSync,
		opCode:     int(so.opCode),
		targetNode: so.targetNode,
		char:       ' ', // SyncOp에서는 char 사용 안 함
	}
}
func (so *OpNodeSyncOp) next() opSequence {
	return so.nextOp
}

// -----------------------------------
// CursorOp 관련 상수 & 노드
// -----------------------------------
type CursorOp int

const (
	OpUpCursor CursorOp = iota
	OpDownCursor
	OpLeftCursor
	OpRightCursor
	OpUpEndCursor
	OpDownEndCursor
	OpRightEndCursor
	OpLeftEndCursor
	OpHoldCursor
)

// OpNodeCursorOp: opSequence 구현체 (Cursor 연산)
type OpNodeCursorOp struct {
	opCode CursorOp
	nextOp opSequence
}

func (co *OpNodeCursorOp) op() opInfo {
	return opInfo{
		opKind:     OpKindCursor,
		opCode:     int(co.opCode),
		targetNode: nil, // 커서 이동이니 targetNode는 사용 안 함
		char:       ' ',
	}
}
func (co *OpNodeCursorOp) next() opSequence {
	return co.nextOp
}
