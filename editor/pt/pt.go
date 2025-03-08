package main

import (
	"fmt"
	"strings"
)

// -------------------------------------
// (1) 기본 구조체 정의 (변경 없음)
// -------------------------------------

type BufferKind int

const (
	BufferOriginal BufferKind = iota
	BufferAdd
)

type Piece struct {
	Kind   BufferKind
	Start  int
	Length int
}

type PieceTable struct {
	originalBuffer []rune
	addBuffer      []rune
	pieces         []Piece
}

func NewPieceTable(initialText string) *PieceTable {
	pt := &PieceTable{
		originalBuffer: []rune(initialText),
		addBuffer:      make([]rune, 0),
		pieces:         make([]Piece, 0),
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

// findPieceAtRuneIndex: 문서상의 0-based 인덱스에 해당하는 piece와 내부 오프셋 반환
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
// (2) Insert(index, text) 수정
//     - 사용자 index를 “index+1” 해서 실제 오프셋으로 해석
// -------------------------------------

func (pt *PieceTable) Insert(index int, newText string) {
	// 1) 실제 삽입 위치 = index + 1
	//    "index번째 문자 **오른쪽**"을 의미
	realOffset := index + 1

	// 경계 보정: 만약 realOffset이 0보다 작거나 큰 경우
	if realOffset < 0 {
		realOffset = 0
	}
	if realOffset > pt.Length() {
		realOffset = pt.Length()
	}

	// 새 텍스트 addBuffer에 추가
	newRunes := []rune(newText)
	startPosInAdd := len(pt.addBuffer)
	pt.addBuffer = append(pt.addBuffer, newRunes...)

	newPiece := Piece{
		Kind:   BufferAdd,
		Start:  startPosInAdd,
		Length: len(newRunes),
	}

	// 만약 realOffset == 문서 끝이면, 그냥 뒤에 붙임
	if realOffset == pt.Length() {
		pt.pieces = append(pt.pieces, newPiece)
		return
	}

	// 삽입할 pieceIndex 찾기
	pieceIndex, internalOffset := pt.findPieceAtRuneIndex(realOffset)
	oldPiece := pt.pieces[pieceIndex]

	frontLength := internalOffset
	backLength := oldPiece.Length - internalOffset

	// (a) 앞부분
	oldPiece.Length = frontLength
	if frontLength == 0 {
		// 앞부분이 0이면 기존 piece를 제거
		pt.pieces = append(pt.pieces[:pieceIndex], pt.pieces[pieceIndex+1:]...)
		pieceIndex--
	} else {
		pt.pieces[pieceIndex] = oldPiece
	}

	// (b) 새 piece 삽입
	pt.pieces = insertPiece(pt.pieces, newPiece, pieceIndex+1)
	pieceIndex++

	// (c) 뒷부분
	if backLength > 0 {
		backPiece := Piece{
			Kind:   oldPiece.Kind,
			Start:  oldPiece.Start + internalOffset,
			Length: backLength,
		}
		pt.pieces = insertPiece(pt.pieces, backPiece, pieceIndex+1)
	}
}

// -------------------------------------
// (3) Delete(start, length) 수정
//     - start = "start번째 문자 **오른쪽**"부터 length개 삭제
//       → realStart = start + 1
// -------------------------------------

func (pt *PieceTable) Delete(start, length int) {
	if length < 0 {
		panic("Delete length < 0")
	}
	// 실제 시작 위치 = start + 1
	realStart := start + 1

	if realStart < 0 {
		realStart = 0
	}
	end := realStart + length
	if end > pt.Length() {
		panic("Delete range out of bounds")
	}

	pieceIndex, offsetInPiece := pt.findPieceAtRuneIndex(realStart)

	var processed int
	for processed < length && pieceIndex < len(pt.pieces) {
		piece := pt.pieces[pieceIndex]
		pieceLen := piece.Length

		canDelete := pieceLen - offsetInPiece
		remain := length - processed
		if remain < canDelete {
			canDelete = remain
		}

		frontLen := offsetInPiece
		backLen := pieceLen - offsetInPiece - canDelete

		if frontLen == 0 && backLen == 0 {
			// 통째로 삭제
			pt.pieces = append(pt.pieces[:pieceIndex], pt.pieces[pieceIndex+1:]...)
			// pieceIndex 증가 안 함
		} else if frontLen == 0 {
			// 뒷부분만 남김
			piece.Start += canDelete
			piece.Length = backLen
			pt.pieces[pieceIndex] = piece
			pieceIndex++
		} else if backLen == 0 {
			// 앞부분만 남김
			piece.Length = frontLen
			pt.pieces[pieceIndex] = piece
			pieceIndex++
		} else {
			// 앞뒤가 다 남는 경우 => 두 덩이로 분할
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
		offsetInPiece = 0
	}
}

// -------------------------------------
// 내부 슬라이스 삽입 유틸 함수
// -------------------------------------
func insertPiece(pieces []Piece, newPiece Piece, idx int) []Piece {
	if idx < 0 {
		idx = 0
	}
	if idx > len(pieces) {
		idx = len(pieces)
	}
	pieces = append(pieces, Piece{})   // 공간 확보
	copy(pieces[idx+1:], pieces[idx:]) // 뒤로 밀기
	pieces[idx] = newPiece
	return pieces
}

// -------------------------------------
// (4) 시연 예제 main
// -------------------------------------
func main() {
	pt := NewPieceTable("Hello World!")
	fmt.Printf("[초기] %q (길이: %d)\n", pt.String(), pt.Length())

	// 예1) Insert(0, ",") => "0번 문자 오른쪽" => 실제 realOffset=1 => 즉 H|ello World! 사이
	pt.Insert(0, ",")
	fmt.Printf("[Insert(0, \",\")] => %q\n", pt.String())

	// 예2) Insert(5, "X") => "5번 문자 오른쪽"
	//     "Hello "에서 ' '이 index=5 (H=0 e=1 l=2 l=3 o=4 ' '=5)
	//     실제 realOffset=6 => 그 뒤에 삽입
	pt.Insert(5, "X")
	fmt.Printf("[Insert(5, \"X\")] => %q\n", pt.String())

	// 예3) Delete(0, 1) => "0번 문자 오른쪽부터 1글자" => 실제 realStart=1 => index=1에 해당하는 'e' 삭제
	pt.Delete(0, 1)
	fmt.Printf("[Delete(0,1)] => %q\n", pt.String())

	// 예4) 문장 끝에 " :)" 추가
	//     끝의 마지막 문자는 index= pt.Length()-1
	//     "끝보다 한 칸 더"면 insert(pt.Length(), ...)
	pt.Insert(pt.Length(), " :)")
	fmt.Printf("[Insert(pt.Length(), \" :)\")] => %q\n", pt.String())
}
