package main

import (
	"strings"
)

// -------------------------------------
// (1) 기본 구조체 정의
// -------------------------------------

type PieceTable struct {
	originalBuffer []rune
	addBuffer      []rune
	pieces         []Piece
	parent         *PieceTable // 히스토리(Undo/Redo) 지원을 위한 원본 참조
}

type Piece struct {
	Kind   BufferKind
	Start  int
	Length int
}

type BufferKind int

const (
	BufferOriginal BufferKind = iota
	BufferAdd
)

// 새로운 PieceTable 생성 (초기 텍스트)
func NewPieceTable(initialText string) *PieceTable {
	pt := &PieceTable{
		originalBuffer: []rune(initialText),
		addBuffer:      make([]rune, 0),
		pieces:         make([]Piece, 0),
		parent:         nil, // 초기 생성은 부모 없음
	}
	if len(pt.originalBuffer) > 0 {
		pt.pieces = append(pt.pieces, Piece{
			Kind:   BufferOriginal,
			Start:  0,
			Length: len(pt.originalBuffer),
		})
	}
	return pt
}

func (pt *PieceTable) Length() int {
	total := 0
	for _, p := range pt.pieces {
		total += p.Length
	}
	return total
}

func (pt *PieceTable) String() string {
	var sb strings.Builder
	for _, piece := range pt.pieces {
		switch piece.Kind {
		case BufferOriginal:
			sb.WriteString(string(pt.originalBuffer[piece.Start : piece.Start+piece.Length]))
		case BufferAdd:
			sb.WriteString(string(pt.addBuffer[piece.Start : piece.Start+piece.Length]))
		}
	}
	return sb.String()
}

// findPieceAtRuneIndex: 문서상의 0-based 인덱스에 해당하는 piece와 내부 offset 반환
func (pt *PieceTable) findPieceAtRuneIndex(index int) (pieceIndex int, internalOffset int) {
	if index < 0 {
		panic("index cannot be negative")
	}
	var sum int
	for i, piece := range pt.pieces {
		if index < sum+piece.Length {
			return i, index - sum
		}
		sum += piece.Length
	}
	return len(pt.pieces) - 1, pt.pieces[len(pt.pieces)-1].Length
}

// -------------------------------------
// (2) Insert / Delete
//     이제 모두 zero-based로, 정확히 해당 인덱스에서 연산하도록 함
// -------------------------------------

// Insert(index, newText):
// 주어진 index 위치에 newText를 삽입한다.
// 예) "hello"에서 Insert(0, "k") → "khello"
//
//	"hello"에서 Insert(2, "X") → "heXllo" (앞에 삽입)
func (pt *PieceTable) Insert(index int, newText string) {
	// 여기서는 realOffset을 index 그대로 사용 (zero-based)
	realOffset := index
	if realOffset < 0 {
		realOffset = 0
	}
	if realOffset > pt.Length() {
		realOffset = pt.Length()
	}

	newRunes := []rune(newText)
	startPosInAdd := len(pt.addBuffer)
	pt.addBuffer = append(pt.addBuffer, newRunes...)

	newPiece := Piece{
		Kind:   BufferAdd,
		Start:  startPosInAdd,
		Length: len(newRunes),
	}

	// 만약 realOffset가 문서 끝이면 그냥 뒤에 붙임
	if realOffset == pt.Length() {
		pt.pieces = append(pt.pieces, newPiece)
		return
	}

	pieceIndex, internalOffset := pt.findPieceAtRuneIndex(realOffset)
	oldPiece := pt.pieces[pieceIndex]

	frontLength := internalOffset                  // 앞쪽은 index 이전까지
	backLength := oldPiece.Length - internalOffset // 뒷쪽은 index부터 끝까지

	// 앞부분: 만약 frontLength > 0이면 남김
	if frontLength > 0 {
		pt.pieces[pieceIndex].Length = frontLength
	} else {
		// front가 없다면 제거
		pt.pieces = append(pt.pieces[:pieceIndex], pt.pieces[pieceIndex+1:]...)
		pieceIndex--
	}

	// 새 piece 삽입: 정확히 index 위치에 삽입됨
	pt.pieces = insertPiece(pt.pieces, newPiece, pieceIndex+1)
	pieceIndex++

	// 뒷부분: 남은 뒤쪽이 있다면 새 piece로 삽입
	if backLength > 0 {
		backPiece := Piece{
			Kind:   oldPiece.Kind,
			Start:  oldPiece.Start + internalOffset,
			Length: backLength,
		}
		pt.pieces = insertPiece(pt.pieces, backPiece, pieceIndex+1)
	}
}

// Delete(start, length):
// 주어진 index(start)부터 <-방향으로 length개 문자를 삭제한다.
// 예) "hello"에서 Delete(1,1) → "ello"
//
//	"hello"에서 Delete(2,2) → "llo"
//
// Delete(cursor, length):
// "cursor" 위치에서 왼쪽으로 length개 문자를 삭제한다.
// 즉, [cursor-length, cursor) 범위를 제거.
// 예) "hello"에서 Delete(1,1) → "ello"
// 예) "hello"에서 Delete(2,2) → "llo"
func (pt *PieceTable) Delete(cursor, length int) {
	if length < 0 {
		panic("Delete length < 0")
	}
	// 실제 삭제 범위: [start, end) = [cursor-length, cursor)
	start := cursor - length
	// 만약 start<0이면, 문서 맨 앞에 도달 -> start=0으로 보정
	if start < 0 {
		start = 0
	}
	end := cursor

	if end > pt.Length() {
		panic("Delete range out of bounds")
	}

	toDelete := end - start
	if toDelete <= 0 {
		// 삭제할 범위가 없거나 0이면 그대로 반환
		return
	}

	// 1) 시작 지점의 Piece index & offset 구하기
	pieceIndex, offsetInPiece := pt.findPieceAtRuneIndex(start)

	var processed int
	for processed < toDelete && pieceIndex < len(pt.pieces) {
		piece := pt.pieces[pieceIndex]
		pieceLen := piece.Length

		// 현재 piece에서 지울 수 있는 범위
		canDelete := pieceLen - offsetInPiece
		remain := toDelete - processed
		if remain < canDelete {
			canDelete = remain
		}

		frontLen := offsetInPiece
		backLen := pieceLen - offsetInPiece - canDelete

		switch {
		case frontLen == 0 && backLen == 0:
			// piece 전체가 삭제됨
			pt.pieces = append(pt.pieces[:pieceIndex], pt.pieces[pieceIndex+1:]...)
			// pieceIndex 증가 안 함 (다음 piece가 현재 인덱스로 옴)

		case frontLen == 0:
			// 앞부분 없음, 뒷부분만 남김
			piece.Start += canDelete
			piece.Length = backLen
			pt.pieces[pieceIndex] = piece
			pieceIndex++

		case backLen == 0:
			// 앞부분만 남고, 뒷부분은 없음
			piece.Length = frontLen
			pt.pieces[pieceIndex] = piece
			pieceIndex++

		default:
			// 앞/뒤가 모두 남음 -> piece를 split
			piece.Length = frontLen
			pt.pieces[pieceIndex] = piece
			newPiece := Piece{
				Kind:   piece.Kind,
				Start:  piece.Start + frontLen + canDelete,
				Length: backLen,
			}
			pt.pieces = insertPiece(pt.pieces, newPiece, pieceIndex+1)
			pieceIndex += 2
		}

		processed += canDelete
		offsetInPiece = 0 // 다음 piece부터는 앞부분 소진 없음
	}
}

// insertPiece: 슬라이스 중간에 새 Piece를 삽입하기 위한 유틸 함수
func insertPiece(pieces []Piece, newPiece Piece, idx int) []Piece {
	if idx < 0 {
		idx = 0
	}
	if idx > len(pieces) {
		idx = len(pieces)
	}
	pieces = append(pieces, Piece{})
	copy(pieces[idx+1:], pieces[idx:])
	pieces[idx] = newPiece
	return pieces
}

// -------------------------------------
// (3) 단일 글자 전용 InsertRune / DeleteRune
// -------------------------------------

func (pt *PieceTable) InsertRune(index int, r rune) {
	// 단일 rune을 문자열로 변환하여 Insert 호출
	pt.Insert(index, string(r))
}

func (pt *PieceTable) DeleteRune(index int) {
	pt.Delete(index, 1)
}

// -------------------------------------
// (4) SlicePieceTable
//     주어진 index(0-based)를 기준으로, 앞쪽은 [0,index) (index 미포함),
//     뒷쪽은 [index, end)로 분할한다.
//     예) "hello"에서 SlicePieceTable(0) → front: "", back: "hello"
//         "hello"에서 SlicePieceTable(1) → front: "h", back: "ello"
// -------------------------------------

func (pt *PieceTable) SlicePieceTable(index int) (*PieceTable, *PieceTable) {
	if index < 0 || index > pt.Length() {
		panic("SlicePieceTable: Index out of range")
	}

	frontPT := &PieceTable{
		originalBuffer: pt.originalBuffer,
		addBuffer:      pt.addBuffer,
		pieces:         []Piece{},
		parent:         pt,
	}
	backPT := &PieceTable{
		originalBuffer: pt.originalBuffer,
		addBuffer:      pt.addBuffer,
		pieces:         []Piece{},
		parent:         pt,
	}

	// 여기서는 index를 포함하지 않으므로,
	// frontPT는 [0,index)이고, backPT는 [index, end)이다.
	pieceIndex, offsetInPiece := pt.findPieceAtRuneIndex(index)

	for i, piece := range pt.pieces {
		if i < pieceIndex {
			frontPT.pieces = append(frontPT.pieces, piece)
		} else if i > pieceIndex {
			backPT.pieces = append(backPT.pieces, piece)
		} else {
			// 현재 분할 대상 Piece:
			// front는 앞부분 [piece.Start, piece.Start+offsetInPiece)
			// back은 뒷부분 [piece.Start+offsetInPiece, piece.Start+piece.Length)
			frontPiece := Piece{
				Kind:   piece.Kind,
				Start:  piece.Start,
				Length: offsetInPiece,
			}
			backPiece := Piece{
				Kind:   piece.Kind,
				Start:  piece.Start + offsetInPiece,
				Length: piece.Length - offsetInPiece,
			}

			if frontPiece.Length > 0 {
				frontPT.pieces = append(frontPT.pieces, frontPiece)
			}
			if backPiece.Length > 0 {
				backPT.pieces = append(backPT.pieces, backPiece)
			}
		}
	}

	return frontPT, backPT
}

// -------------------------------------
// (5) 테스트 코드
// -------------------------------------

// func main() {
// 	// 기본 테스트: "Hello World!"
// 	pt := NewPieceTable("Hello World!")
// 	fmt.Println("[초기] ", pt.String()) // "Hello World!"

// 	// 1) InsertRune(0, 'k') -> "kHello World!"
// 	pt.InsertRune(0, 'k')
// 	fmt.Println("[InsertRune(0, 'k')]", pt.String())

// 	// 2) DeleteRune(0) -> "Hello World!"
// 	pt.DeleteRune(0)
// 	fmt.Println("[DeleteRune(0)] ", pt.String())

// 	// 3) InsertRune(4, 'Z') -> "HellZo World!" (인덱스 4 앞에 'Z' 삽입)
// 	pt.InsertRune(4, 'Z')
// 	fmt.Println("[InsertRune(4, 'Z')]", pt.String())

// 	// 4) DeleteRune(10) -> 인덱스 10 문자 삭제
// 	pt.DeleteRune(10)
// 	fmt.Println("[DeleteRune(10)] ", pt.String())

// 	// 5) SlicePieceTable 테스트
// 	// "Hello World!"에서 SlicePieceTable(0): front "", back "Hello World!"
// 	pt = NewPieceTable("Hello World!")
// 	front, back := pt.SlicePieceTable(0)
// 	fmt.Println("[SlicePieceTable(0)] Front:", front.String(), "Back:", back.String())

// 	// "Hello World!"에서 SlicePieceTable(1): front "H", back "ello World!"
// 	front, back = pt.SlicePieceTable(1)
// 	fmt.Println("[SlicePieceTable(1)] Front:", front.String(), "Back:", back.String())

// 	// "Hello World!"에서 SlicePieceTable(6): front "Hello ", back "World!"
// 	front, back = pt.SlicePieceTable(6)
// 	fmt.Println("[SlicePieceTable(6)] Front:", front.String(), "Back:", back.String())
// }
