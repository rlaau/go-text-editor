package syncer

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

	//커서 위치 0,0으로 이동
	sp.cursor.currentLineBuffer = sp.syncData.head.LineBuffer
	sp.cursor.currentCharInset = 0
	println("lineCount", lineCount, "만큼 초기화")
	return sp
}

// ProcessCommand는 에디터에서 최종 호출해서 명령어 처리함
func (sp *SyncProtocol) ProcessCommand(cmd commander.Command) (
	isContinue bool) {
	opSequences, isContinue := sp.buildOpSequences(cmd)
	if !isContinue {
		return false
	}

	opSequences.ExecuteAll(sp)

	return true

}

// interpretCommand는 시퀀스 및 성공여부 리턴턴
func (sp *SyncProtocol) buildOpSequences(cmd commander.Command) (*opSequences, bool) {

	ops := NewOpSequences()
	cursorLine, _ := sp.cursor.GetCoordinate()
	currentNode := sp.syncData.findSyncNodeByLineBuffer(cursorLine)

	//노드가 있다면 피스테이블은 항상 같이 존재함
	//그러나 라인버퍼는 존재를 모르므로 항상 주의
	var opNodeGroup *OpNodeGroup
	var opNodeText *OpNodeText
	var opSync *OpSync
	var opCusror *OpCursor
	textLen := currentNode.PieceTable.Length()
	switch cmd.Code {
	case commander.CmdExit:
		return nil, false
	case commander.CmdDelete:
		if textLen == 0 {
			//텍스트가 0인데 맨 위인 경우
			//그냥 그대로 둚
			if currentNode.IsUpperEnd() {
				opNodeGroup, opNodeText, opSync, opCusror = sp.buildTotalOpSequence(
					OpHoldAllGroup,
					currentNode,
					OpHoldRune,
					rune(' '),
					OpNodeHoldSync,
					currentNode,
					OpHoldCursor)
			} else {
				//커서를 미리 해당 라인으로 한칸 올려 둔 후에 오른쪽 끝으로 옮기기
				sp.cursor.currentLineBuffer = currentNode.prev.LineBuffer
				//맨 위까진 아녀서 하나 지우고 올라가는 경우
				opNodeGroup, opNodeText, opSync, opCusror = sp.buildTotalOpSequence(
					OpDeletNodeFromGroup,
					currentNode,
					OpHoldRune,
					rune(' '),
					OpNodeDeletedSync,
					currentNode,
					OpRightEndCursor)

			}
		} else if sp.cursor.currentCharInset == 0 {
			//딜리트를 했는데 그게 맨 시작인 경우
			if currentNode.IsUpperEnd() {
				//아무것도 안하기
				opNodeGroup, opNodeText, opSync, opCusror = sp.buildTotalOpSequence(
					OpHoldAllGroup,
					currentNode,
					OpHoldRune,
					rune(' '),
					OpNodeHoldSync,
					currentNode,
					OpHoldCursor)
			} else {
				prevNode := currentNode.prev
				//미리 위로 한칸 올라가기. 글고 위치 자체를 미리 지정
				sp.cursor.currentLineBuffer = prevNode.LineBuffer
				sp.cursor.currentCharInset = prevNode.PieceTable.Length()
				opNodeGroup, opNodeText, opSync, opCusror = sp.buildTotalOpSequence(
					OpMergeNodesInGroup,
					prevNode,
					OpHoldRune,
					rune(' '),
					OpNodeModifiedSync,
					prevNode,
					OpHoldCursor,
				)
			}

		} else {
			opNodeGroup, opNodeText, opSync, opCusror = sp.buildTotalOpSequence(
				OpModifyNodeOnGroup,
				currentNode,
				OpDeleteRune,
				rune(' '),
				OpNodeModifiedSync,
				currentNode,
				OpLeftCursor,
			)
		}
	case commander.CmdInsert:
		if charInput, ok := cmd.Input.(commander.CharInput); ok {
			var co CursorOpCode
			if currentNode.IsDownEnd() {
				co = OpHoldCursor
			} else {
				co = OpDownLeftStartCursor
			}
			//엔터키 눌린 경우
			if charInput.Char == commander.KeyEnter1 || charInput.Char == commander.KeyEnter2 {

				opNodeGroup, opNodeText, opSync, opCusror = sp.buildTotalOpSequence(
					OpSliceNodeAtGroup,
					currentNode,
					OpHoldRune,
					rune(' '),
					OpNodeSlicedSync,
					currentNode,
					co)
			} else {
				//그냥 문자열 인서트였을 경우
				opNodeGroup, opNodeText, opSync, opCusror = sp.buildTotalOpSequence(
					OpModifyNodeOnGroup,
					currentNode,
					OpInsertRune,
					charInput.Char,
					OpNodeModifiedSync,
					currentNode,
					OpRightCursor,
				)

			}
		}
	case commander.CmdMove:
		//커서 움직임만 있는 경우 노드 및 데이터의 변경이 존재 x
		opCusror = sp.buildCursorOp(cmd)
	}

	if opNodeGroup != nil {
		ops.Append(opNodeGroup)
	}
	if opNodeText != nil {
		ops.Append(opNodeText)
	}
	if opSync != nil {
		ops.Append(opSync)
	}
	if opCusror != nil {
		ops.Append(opCusror)
	}
	return ops, true
}
func (sp *SyncProtocol) buildTotalOpSequence(ngo NodeGroupOpCode, ngsn *SyncNode,
	nto NodeTextOpCode, char rune,
	so SyncOpCode, ssn *SyncNode,
	cursorOp CursorOpCode) (*OpNodeGroup, *OpNodeText, *OpSync, *OpCursor) {
	return NewOpNodeGroup(ngo, ngsn),
		NewOpNodeText(nto, char),
		NewOpNodeSync(so, ssn),
		NewOpNodeCursor(cursorOp)

}

func (sp *SyncProtocol) buildCursorOp(cmd commander.Command) *OpCursor {
	var opCursorCode CursorOpCode
	cursorLine, charInset := sp.cursor.GetCoordinate()
	//주의 할 것!! 커서는 항상 라인버퍼에 자신을 동기화함!
	//그러나 이 경우엔 커서만 움직일 경우 상관 x
	syncNode := sp.syncData.findSyncNodeByLineBuffer(cursorLine)
	textLen := syncNode.PieceTable.Length()
	if charInput, ok := cmd.Input.(commander.CharInput); ok {
		switch charInput.Char {
		case commander.KeyUp:
			if syncNode.IsUpperEnd() {
				opCursorCode = OpHoldCursor
			} else {
				opCursorCode = OpUpCursor
			}
		case commander.KeyDown:
			if syncNode.IsDownEnd() {
				opCursorCode = OpHoldCursor
			} else {
				opCursorCode = OpDownCursor
			}
		case commander.KeyLeft:
			if textLen == 0 || sp.cursor.currentCharInset == 0 {
				if syncNode.IsUpperEnd() {
					opCursorCode = OpHoldCursor
				} else {
					opCursorCode = OpUpCursor
				}
			} else {
				opCursorCode = OpLeftCursor
			}
		case commander.KeyRight:
			if charInset >= textLen {
				if syncNode.IsDownEnd() {
					opCursorCode = OpHoldCursor
				} else {
					opCursorCode = OpDownLeftStartCursor
				}
			} else {
				opCursorCode = OpRightCursor
			}
		}
	}
	sp.cursor.visible = true
	return NewOpNodeCursor(opCursorCode)
}

func (sp *SyncProtocol) syncNode(sn *SyncNode) {
	//TODO 추후 반드시 커서의 위치는 라인버퍼가 아닌 싱크노드로 하기!!!!!!
	//TODO 반드시!!!!!!!!!!!!!!!!!
	str := sn.PieceTable.String()
	if sn.LineBuffer == nil {
		needToMove := false
		if sp.syncData.findOrder(sn) == sp.syncData.findOrder(sp.syncData.findSyncNodeByLineBuffer(sp.cursor.currentLineBuffer)) {
			needToMove = true
		}

		sn.LineBuffer = sp.NewLineBuffer() // LineBuffer를 생성하고 SyncNode에 설정

		if needToMove {
			sp.cursor.currentLineBuffer = sn.LineBuffer
		}
	}

	sp.ReflectLine(sn.LineBuffer, str)
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

func (sn *SyncNode) IsUpperEnd() bool {
	return sn.prev == sn || sn.prev == nil
}
func (sn *SyncNode) IsDownEnd() bool {
	return sn.next == sn || sn.next == nil
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
	if n == 0 {
		sd.insertByPtr(nil, newData) // 빈 리스트 또는 맨 앞에 삽입
		return
	}
	cur, found := sd.findNode(n - 1) // n-1 번째 노드 뒤에 삽입
	if !found {
		return
	}
	sd.insertByPtr(cur, newData)
}

// deleteNode(n): n번째 노드를 삭제 (0-based)
func (sd *SyncData) deleteNode(n uint) {

	cur, found := sd.findNode(n)
	if !found {
		return
	}
	sd.deleteByPtr(cur)

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
	sd.sliceByPtr(cur, char)

}

// insertByPtr(refNode, newData):
// refNode "앞"에 새 노드를 삽입.
func (sd *SyncData) insertByPtr(refNode *SyncNode, newData string) {
	newNode := &SyncNode{
		PieceTable: NewPieceTable(newData),
		LineBuffer: nil, // 이후 SyncProtocol.syncNode에서 설정됨
		prev:       nil,
		next:       nil,
	}

	if refNode == nil {
		if sd.head == nil {
			sd.head = newNode
		} else {
			newNode.next = sd.head
			sd.head.prev = newNode
			sd.head = newNode
		}
	} else {
		newNode.next = refNode
		newNode.prev = refNode.prev
		if refNode.prev != nil {
			refNode.prev.next = newNode
		} else {
			sd.head = newNode
		}
		refNode.prev = newNode
	}
}

// deleteByPtr(refNode):
// refNode를 이중 연결 리스트에서 제거.
func (sd *SyncData) deleteByPtr(refNode *SyncNode) {
	if refNode == nil || sd.head == nil {
		return
	}

	// head 삭제 처리
	if refNode == sd.head {
		sd.head = refNode.next
		if sd.head != nil {
			sd.head.prev = nil
		}
	} else {
		// 중간(또는 tail) 노드 제거
		if refNode.prev != nil {
			refNode.prev.next = refNode.next
		}
		if refNode.next != nil {
			refNode.next.prev = refNode.prev
		}
	}

	// 참조 해제
	refNode.prev = nil
	refNode.next = nil
}

// sliceByPtr(refNode, char):
// refNode.PieceTable을 앞뒤로 분리하여 refNode 앞부분 유지, 뒷부분을 새로운 노드로 삽입.
func (sd *SyncData) sliceByPtr(refNode *SyncNode, char uint) {
	if refNode == nil || refNode.PieceTable == nil {
		return
	}

	frontPT, backPT := refNode.PieceTable.SlicePieceTable(int(char))
	if frontPT == nil || backPT == nil {
		return
	}
	refNode.PieceTable = frontPT // 기존 노드는 앞부분 유지

	// 새로운 노드 생성 및 연결
	newNode := &SyncNode{
		PieceTable: backPT,
		LineBuffer: nil,
		prev:       refNode,
		next:       refNode.next,
	}
	if refNode.next != nil {
		refNode.next.prev = newNode
	}
	refNode.next = newNode

	// **새 노드가 없으면 head 갱신**
	if sd.head == nil {
		sd.head = newNode
	}
}

// mergeNodeByPtr은 아예 새 문자열 생성하는 식으로 머징
// prev를 살리는 방식으로 머지합니다.
func (sd *SyncData) mergeNodeByPtr(prev, next *SyncNode) {
	if prev == nil || next == nil || prev.next != next || next.prev != prev {
		return
	}

	prevStr := prev.PieceTable.String()
	nextStr := next.PieceTable.String()
	mergedStr := prevStr + nextStr
	prev.PieceTable = NewPieceTable(mergedStr)

	if next.next != nil {
		next.next.prev = prev
	}
	prev.next = next.next

	next.prev = nil
	next.next = nil

	if sd.head == next {
		sd.head = prev
	}

	// 호출자가 SyncProtocol 인스턴스에 접근할 수 있도록 외부에서 syncNode 호출 필요
	// 또는 SyncProtocol을 매개변수로 받아 내부에서 호출 가능:
	// sp.syncNode(prev)
}

// modifyByPtr(refNode, cursorChar, char, modifyCode):
// refNode의 PieceTable을 수정.
func (sd *SyncData) modifyByPtr(refNode *SyncNode, cursorChar int, char rune, modifyCode ModifyCode) {
	if refNode == nil || refNode.PieceTable == nil {
		return
	}

	switch modifyCode {
	case DeleteASCII:
		refNode.PieceTable.DeleteRune(cursorChar)
	case InsertASCII:
		refNode.PieceTable.InsertRune(cursorChar, char)
	}
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

func (sd *SyncData) findOrder(sn *SyncNode) int {
	if sd.head == nil {
		return -1
	}
	cur := sd.head
	count := uint(0)
	for cur.next != nil {
		if cur == sn {
			return int(count)
		}
		cur = cur.next
		count++
	}
	if cur == sn {
		return int(count)
	}
	return -1

}

// 그리고, 두 함수를 이 findSyncNode에 위임

func (sd *SyncData) findSyncNodeByLineBuffer(lb *LineBuffer) *SyncNode {
	return sd.findSyncNode(func(sn *SyncNode) bool {
		return sn.LineBuffer == lb
	})
}

func (sd *SyncData) findSyncNodeByPieceTable(pt *PieceTable) *SyncNode {
	return sd.findSyncNode(func(sn *SyncNode) bool {
		return sn.PieceTable == pt
	})
}

func (sd *SyncData) findSyncNode(predicate func(*SyncNode) bool) *SyncNode {
	var found *SyncNode
	sd.ForEach(func(sn *SyncNode) {
		// 아직 찾은 게 없고, 조건이 맞으면 found에 할당
		if found == nil && predicate(sn) {
			found = sn
		}
	})
	return found
}
