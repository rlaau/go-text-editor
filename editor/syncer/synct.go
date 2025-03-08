package main

import (
	"fmt"
)

// (4) 테스트 코드
// -----------------------------------------------------

// 테스트용 main 함수
func main() {
	// 1) 프로토콜 생성
	sp := NewSyncProtocol(800, 600, 0xFF000000, 0xFFFFFFFF)

	// 2) 초기 노드 2개 생성
	// 첫 번째 노드: "hello"
	sp.syncData.insertNode(0, "hello") // 빈 리스트이므로 head가 "hello"가 됨.
	// 두 번째 노드: "world"를 0번 위치에 삽입 → 새 노드가 head로 추가되어, 순서가 "world", "hello"가 됨.
	sp.syncData.insertNode(0, "world")

	// 현재 상태: head -> [ "world" ] <-> [ "hello" ] -> nil
	fmt.Println("== 초기 리스트 ==")
	printList(sp.syncData.head)

	// 3) 첫 번째 노드("world")에서 첫 번째 'r'의 위치를 기준으로 슬라이스
	//    예를 들어 "world"에서 인덱스 2에 해당하는 문자('r') 전까지 분할하면,
	//    front: "wo", back: "rld"
	sp.syncData.sliceNode(0, 2)
	fmt.Println("\n== sliceNode(0, 2) 호출 후 ==")
	printList(sp.syncData.head)

	// 4) 두 번째 노드("rld")를 삭제해보기 (0-based, 두 번째 노드는 index 1)
	sp.syncData.deleteNode(1)
	fmt.Println("\n== delete(1) 호출 후 (이중 연결 리스트) ==")
	printList(sp.syncData.head)

	sp.syncData.modifyNode(0, 1, rune('k'), InsertASCII)
	fmt.Println("\n== 모디파이1")
	printList(sp.syncData.head)

	sp.syncData.modifyNode(0, 3, rune(' '), DeleteASCII)
	fmt.Println("\n== 모디파이2")
	printList(sp.syncData.head)

	// 5) 남은 리스트 역순으로 확인 (이중 연결 리스트)
	fmt.Println("\n== 역순 확인 ==")
	printListReverse(sp.syncData.head)
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
