package main

import (
	"fmt"
	"go_editor/editor/commander"
)

// (4) 테스트 코드
// -----------------------------------------------------

// 테스트용 main 함수
func main() {
	// 1) 프로토콜 생성
	sp := NewSyncProtocol(800, 600, 0xFF000000, 0xFFFFFFFF, 16)

	// 2) 초기 노드 2개 생성 (예시)
	// 첫 번째 노드: "hello"
	sp.syncData.insertNode(0, "hello")
	// 두 번째 노드: "world"
	sp.syncData.insertNode(1, "world")
	// 두 번째 노드의 PieceTable에 'w' 추가 예시
	sp.syncData.modifyNode(1, 1, rune('w'), InsertASCII)

	fmt.Println("== 초기 리스트 ==")
	printList(sp.syncData.head)

	// 3) 첫 번째 노드("world")에서 인덱스 2를 기준으로 슬라이스
	//    ("wo" | "rld") 예시
	sp.syncData.sliceNode(0, 2)
	fmt.Println("\n== sliceNode(0, 2) 호출 후 ==")
	printList(sp.syncData.head)

	// 4) 두 번째 노드("llo")를 삭제 (0-based: index=1)
	sp.syncData.deleteNode(1)
	fmt.Println("\n== delete(1) 호출 후 (이중 연결 리스트) ==")
	printList(sp.syncData.head)

	sp.syncData.modifyNode(0, 1, rune('k'), InsertASCII)
	fmt.Println("\n== 모디파이1")
	printList(sp.syncData.head)

	sp.syncData.modifyNode(0, 1, rune(' '), DeleteASCII)
	fmt.Println("\n== 모디파이2")
	printList(sp.syncData.head)

	// 5) 남은 리스트 역순 확인
	fmt.Println("\n== 역순 확인 ==")
	printListReverse(sp.syncData.head)

	// -----------------------------------------------------
	// (추가) 커맨더 명령(Command) 직접 만들어서 테스트하기
	// -----------------------------------------------------
	fmt.Println("\n== Command Processing Test ==")

	// 예시로 8개 명령어 준비 (각 케이스당 2개씩)
	testCommands := []commander.Command{
		// 1) "마우스무브(CmdMove)" - 실제 구현은 화살표 이동으로 처리
		//    KeyLeft, KeyRight 를 CharInput으로 전달
		{Code: commander.CmdMove, Input: commander.CharInput{Char: commander.KeyLeft}},
		{Code: commander.CmdMove, Input: commander.CharInput{Char: commander.KeyRight}},

		// 2) "인서트(CmdInsert)" - 문자 'A', 그리고 엔터
		{Code: commander.CmdInsert, Input: commander.CharInput{Char: 'A'}},
		{Code: commander.CmdInsert, Input: commander.CharInput{Char: commander.KeyEnter1}},

		// 3) "딜리트(CmdDelete)" - 백스페이스(두 번)
		{Code: commander.CmdDelete, Input: commander.CharInput{Char: commander.KeyBackSpace}},
		{Code: commander.CmdDelete, Input: commander.CharInput{Char: commander.KeyBackSpace}},

		// 4) "엑시트(CmdExit)" - ESC를 두 번
		{Code: commander.CmdExit, Input: commander.CharInput{Char: commander.KeyESC}},
		{Code: commander.CmdExit, Input: commander.CharInput{Char: commander.KeyESC}},
	}
	println()
	for i, cmd := range testCommands {
		fmt.Printf("[Command #%d] -> Code=%v, Input=%v\n", i, cmd.Code, cmd.Input)

		isContinue := sp.ProcessCommand(cmd)
		fmt.Printf("   Processed => isContinue=%v\n", isContinue)

		// 매번 실행 후, 리스트 상태 확인
		printList(sp.syncData.head)

		println("\n")
	}
}

// printList: head부터 next 방향으로 순회하며 출력
func printList(head *SyncNode) {
	if head == nil {
		fmt.Println("Empty list.")
		return
	}
	cur := head
	i := 0
	for cur != nil {
		if cur.PieceTable.String() == "" {
			println("빈 텍스트 발견- 종료")
			return
		}
		fmt.Printf(" Node[%d]: %q\n", i, cur.PieceTable.String())
		cur = cur.next
		i++
	}
}

// printListReverse: tail까지 간 뒤, prev 방향으로 역순 출력 (이중 연결 리스트 확인용)
func printListReverse(head *SyncNode) {
	if head == nil {
		fmt.Println("Empty list.")
		return
	}
	// tail 찾기
	cur := head
	for cur.next != nil {
		cur = cur.next
	}
	i := 0
	for cur != nil {
		fmt.Printf(" ReverseNode[%d]: %q\n", i, cur.PieceTable.String())
		cur = cur.prev
		i++
	}
}
