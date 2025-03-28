package syncer

import "fmt"

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

// [ADDED] ExecuteAll 메서드 예시:
//
//	모든 노드를 순회하면서 각 노드의 executeOp()를 실행하는 예시입니다.
//	필요에 따라 이름과 사용 방식을 조정하세요.
func (ops *opSequences) ExecuteAll(sp *SyncProtocol) {
	current := ops.head
	for current != nil {
		current.executeOp(sp)
		current = current.next()
	}
	println("모든 노드의 executeOp() 실행 완료")
}

// ForEach() : 모든 노드를 순회하며, 각 노드의 opInfo를 fn으로 전달
func (ops *opSequences) ForEach(fn func(info opInfo)) {
	current := ops.head
	for current != nil {
		fn(current.op())
		current = current.next()
	}
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
	case *OpNodeText:
		t.nextOp = seq
	case *OpSync:
		t.nextOp = seq
	case *OpCursor:
		t.nextOp = seq
	}
}

// -----------------------------------
// 아래부터는 기존 코드 (인터페이스, 노드들) 그대로 사용
// -----------------------------------

type opSequence interface {
	op() opInfo
	next() opSequence
	executeOp(*SyncProtocol)
}

type opInfo struct {
	opKind     opKind
	opCode     int
	targetNode *SyncNode
	char       rune
}

type opKind int

const (
	OpKindNodeGroup opKind = iota
	OpKindNodeText
	OpKindSync
	OpKindCursor
)

// -----------------------------------
// NodeGroup 관련 상수 & 노드
// -----------------------------------
type NodeGroupOpCode int

const (
	OpInserNodeToGroup NodeGroupOpCode = iota
	OpDeletNodeFromGroup
	OpSliceNodeAtGroup
	OpModifyNodeOnGroup
	OpMergeNodesInGroup
	OpHoldAllGroup
)

// OpNodeGroup: opSequence 구현체 (NodeGroup 연산)
type OpNodeGroup struct {
	opCode    NodeGroupOpCode
	startNode *SyncNode
	nextOp    opSequence
}

func (ng *OpNodeGroup) op() opInfo {
	return opInfo{
		opKind:     OpKindNodeGroup,
		opCode:     int(ng.opCode),
		targetNode: ng.startNode,
		char:       ' ', // NodeGroup에서는 char 사용 안 함
	}
}
func (ng *OpNodeGroup) next() opSequence {

	if ng.nextOp == nil {
		return nil
	}
	return ng.nextOp
}

// [ADDED] executeOp(): opCode에 따른 실제 동작을 switch로 분기
func (ng *OpNodeGroup) executeOp(sp *SyncProtocol) {
	sd := sp.syncData
	_, charInset := sp.cursor.GetCoordinate()

	switch ng.opCode {
	case OpInserNodeToGroup:
		fmt.Println("NodeGroup -> InsertNodeToGroup 실행")
		sd.insertByPtr(ng.startNode, string(""))
	case OpDeletNodeFromGroup:
		fmt.Println("NodeGroup -> DeleteNodeFromGroup 실행")
		sd.deleteByPtr(ng.startNode)
	case OpSliceNodeAtGroup:
		fmt.Println("NodeGroup -> SliceNodeAtGroup 실행")
		println("시작 노드 번호", sp.syncData.findOrder(ng.startNode))
		sd.sliceByPtr(ng.startNode, uint(charInset))
	case OpMergeNodesInGroup:
		fmt.Println("NodeGroup -> MergeNodesInGroup 실행")
		sd.mergeNodeByPtr(ng.startNode, ng.startNode.next)
	case OpModifyNodeOnGroup:
		fmt.Println("NodeGroup -> ModifyNodeOnGroup 실행")
		//여기선 일단 홀딩
	case OpHoldAllGroup:
		//여기도 일단 홀드
		fmt.Println("NodeGroup -> HoldAllGroup 실행")
	default:
		fmt.Println("NodeGroup -> 알 수 없는 opCode")
	}
}
func NewOpNodeGroup(code NodeGroupOpCode, target *SyncNode) *OpNodeGroup {
	return &OpNodeGroup{
		opCode:    code,
		startNode: target,
	}
}

// -----------------------------------
// NodeText 관련 상수 & 노드
// -----------------------------------
type NodeTextOpCode int

const (
	OpInsertRune NodeTextOpCode = iota
	OpDeleteRune
	OpHoldRune
)

// OpNodeText: opSequence 구현체 (NodeText 연산)
type OpNodeText struct {
	opCode NodeTextOpCode
	char   rune
	nextOp opSequence
}

func (nt *OpNodeText) op() opInfo {
	return opInfo{
		opKind:     OpKindNodeText,
		opCode:     int(nt.opCode),
		targetNode: nil, // 텍스트 관련이라 targetNode는 사용 안 함
		char:       nt.char,
	}
}
func (nt *OpNodeText) next() opSequence {
	if nt.nextOp == nil {
		return nil
	}
	return nt.nextOp
}

// [ADDED] executeOp()
func (nt *OpNodeText) executeOp(sp *SyncProtocol) {
	lineBuffer, charInset := sp.cursor.GetCoordinate()
	syncNode := sp.syncData.findSyncNodeByLineBuffer(lineBuffer)
	switch nt.opCode {
	case OpInsertRune:
		fmt.Printf("NodeText -> InsertRune(%c)\n", nt.char)
		syncNode.PieceTable.InsertRune(charInset, nt.char)
	case OpDeleteRune:
		fmt.Printf("NodeText -> DeleteRune(%c)\n", nt.char)
		//여기선 최대한 간결한 동장을 지향함
		// 가드클로스는 빌딩때 다 처리
		println(charInset, "의 문자 삭제함")
		println("전처리 전", syncNode.PieceTable.String())
		syncNode.PieceTable.DeleteRune(charInset)
		println("전처리 이후", syncNode.PieceTable.String())
	case OpHoldRune:
		fmt.Printf("NodeText -> HoldRune(%c)\n", nt.char)
	default:
		fmt.Println("NodeText -> 알 수 없는 opCode")
	}
}

// [ADDED] 생성자: opCode->실행함수 매핑 주입
func NewOpNodeText(code NodeTextOpCode, ch rune) *OpNodeText {
	return &OpNodeText{
		opCode: code,
		char:   ch,
	}
}

// -----------------------------------
// SyncOpCode 관련 상수 & 노드
// -----------------------------------
type SyncOpCode int

const (
	OpNodeInsertedSync SyncOpCode = iota
	OpNodeSlicedSync
	OpNodeModifiedSync
	OpNodeDeletedSync
	OpNodeHoldSync
)

// OpSync: opSequence 구현체 (Sync 연산)
type OpSync struct {
	opCode    SyncOpCode
	startNode *SyncNode

	nextOp opSequence
}

func (so *OpSync) op() opInfo {
	return opInfo{
		opKind: OpKindSync,
		opCode: int(so.opCode),

		targetNode: so.startNode,
		char:       ' ', // SyncOp에서는 char 사용 안 함
	}
}
func (so *OpSync) next() opSequence {
	if so.nextOp == nil {
		return nil
	}
	return so.nextOp
}

// [ADDED] executeOp()
func (so *OpSync) executeOp(sp *SyncProtocol) {
	switch so.opCode {
	case OpNodeInsertedSync:
		fmt.Println("Sync -> NodeInsertedSync 실행")
		//스타트와 스타트의 prev (단 prev는 nil일수도 있다.)
		prevNode := so.startNode.prev
		if prevNode != nil {
			sp.syncNode(prevNode)
		}
		sp.syncNode(so.startNode)
	case OpNodeSlicedSync:
		fmt.Println("Sync -> NodeSlicedSync 실행")
		//스타트와 스타트의 next (이 경우엔 둘다 반드시 존재.)
		nextNode := so.startNode.next

		if nextNode != nil {
			sp.syncNode(nextNode)
		}
		sp.syncNode(so.startNode)
	case OpNodeModifiedSync:
		fmt.Println("Sync -> NodeModifiedSync 실행")
		//하나만 싱크
		sp.syncNode(so.startNode)
	case OpNodeDeletedSync:
		fmt.Println("Sync -> NodeDeletedSync 실행")
		//일단 암것도 안함
	case OpNodeHoldSync:
		fmt.Println("Sync -> NodeHoldSync 실행")
	default:
		fmt.Println("Sync -> 알 수 없는 opCode")
	}
}

// [ADDED] 생성자: opCode->실행함수 매핑 주입
func NewOpNodeSync(code SyncOpCode, target *SyncNode) *OpSync {
	return &OpSync{
		opCode:    code,
		startNode: target,
	}
}

// -----------------------------------
// CursorOpCode 관련 상수 & 노드
// -----------------------------------
type CursorOpCode int

const (
	OpUpCursor CursorOpCode = iota
	OpDownCursor
	OpLeftCursor
	OpRightCursor
	OpUpLeftStartCursor
	OpUpRightEndCursor
	OpDownLeftStartCursor
	OpDownRightEndCursor
	OpLeftStartCursor
	OpRightEndCursor
	OpHoldCursor
)

// OpCursor: opSequence 구현체 (Cursor 연산)
type OpCursor struct {
	opCode CursorOpCode

	nextOp opSequence
}

func (co *OpCursor) op() opInfo {
	return opInfo{
		opKind: OpKindCursor,
		opCode: int(co.opCode),

		targetNode: nil, // 커서 이동이니 targetNode는 사용 안 함
		char:       ' ',
	}
}
func (co *OpCursor) next() opSequence {
	if co.nextOp == nil {
		return nil
	}
	return co.nextOp
}

// [ADDED] executeOp()
func (co *OpCursor) executeOp(sp *SyncProtocol) {
	c := sp.cursor
	cuerrentLine, currentCharInset := c.GetCoordinate()
	currentNode := sp.syncData.findSyncNodeByLineBuffer(cuerrentLine)
	switch co.opCode {
	case OpUpCursor:
		fmt.Println("Cursor -> UpCursor 실행")
		// 가드 클로스는 빌딩때 처리함
		//라인만 한칸 이동
		prevNode := currentNode.prev
		c.currentLineBuffer = prevNode.LineBuffer
		// 라인 길이를 최대로 삼음
		maxInset := max(currentCharInset, prevNode.PieceTable.Length())
		c.currentCharInset = maxInset
	case OpDownCursor:
		fmt.Println("Cursor -> DownCursor 실행")
		// 가드 클로스는 빌딩때 처리함
		//라인만 한칸 이동
		nextNode := currentNode.next
		c.currentLineBuffer = nextNode.LineBuffer
		// 라인 길이를 최대로 삼음
		maxInset := max(currentCharInset, nextNode.PieceTable.Length())
		c.currentCharInset = maxInset
	case OpLeftCursor:
		fmt.Println("Cursor -> LeftCursor 실행")
		// 가드 클로스는 빌딩때 처리함
		c.currentCharInset = max(c.currentCharInset-1, 0)
	case OpRightCursor:
		fmt.Println("Cursor -> RightCursor 실행")
		// 가드 클로스는 빌딩때 처리함
		c.currentCharInset += 1
	case OpUpLeftStartCursor:
		fmt.Println("Cursor -> UpLeftStartCursor 실행")
		prevNode := currentNode.prev
		c.currentLineBuffer = prevNode.LineBuffer
		c.currentCharInset = 0
	case OpUpRightEndCursor:
		fmt.Println("Cursor -> UpRightEndCursor 실행")
		prevNode := currentNode.prev
		c.currentLineBuffer = prevNode.LineBuffer
		c.currentCharInset = prevNode.PieceTable.Length()
	case OpDownLeftStartCursor:
		fmt.Println("Cursor -> DownLeftStartCursor 실행")
		nextNode := currentNode.next
		c.currentLineBuffer = nextNode.LineBuffer
		c.currentCharInset = 0
	case OpDownRightEndCursor:
		fmt.Println("Cursor -> DownRightEndCursor 실행")
		nextNode := currentNode.next
		c.currentLineBuffer = nextNode.LineBuffer
		c.currentCharInset = nextNode.PieceTable.Length()
	case OpLeftStartCursor:
		fmt.Println("Cursor -> LeftStartCursor 실행")
		c.currentCharInset = 0
	case OpRightEndCursor:
		fmt.Println("Cursor -> RightEndCursor 실행")
		c.currentCharInset = currentNode.PieceTable.Length()
	case OpHoldCursor:
		fmt.Println("Cursor -> HoldCursor 실행")
	default:
		fmt.Println("Cursor -> 알 수 없는 opCode")
	}
}

// [ADDED] 생성자: opCode->실행함수 매핑 주입
func NewOpNodeCursor(code CursorOpCode) *OpCursor {
	return &OpCursor{
		opCode: code,
	}
}
