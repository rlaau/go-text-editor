package main

import "fmt"

func main() {
	// 1) 프로토콜 생성
	sp := NewSyncProtocol()

	// 2) 초기 노드 2개 생성
	//    첫 번째 노드: "hello"
	sp.syncData.insert(0, "hello") // 리스트가 비어있으므로 첫 노드가 됨
	//    두 번째 노드: "world" (첫 번째 노드 뒤에)
	sp.syncData.insert(0, "world")

	// 현재 상태: head -> [ "hello" ] <-> [ "world" ] -> nil
	fmt.Println("== 초기 리스트 ==")
	printList(sp.syncData.head)

	// 3) 첫 번째 노드("hello")에서 첫 번째 'l' 위치(index=2)로 슬라이스

	sp.syncData.sliceNode(0, 5)
	fmt.Println("\n== SliceNode(0, 5) 호출 후 ==")
	printList(sp.syncData.head)

	// 4) 두 번째 노드("llo")를 삭제해보기
	sp.syncData.delete(2)
	fmt.Println("\n== Delete(2) 호출 후 (이중 연결 리스트) ==")
	printList(sp.syncData.head)

	// 5) 남은 리스트 ( "he" <-> "world" )
	fmt.Println("\n== 역순으로도 확인(이중 연결) ==")
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
		fmt.Printf(" Node[%d]: %q\n", i, cur.PieceTable.data)
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
	// tail 찾아가기
	cur := head
	for cur.next != nil {
		cur = cur.next
	}
	// 이제 cur는 tail
	i := 1
	for cur != nil {
		fmt.Printf(" ReverseNode[%d]: %q\n", i, cur.PieceTable.data)
		cur = cur.prev
		i++
	}
}
