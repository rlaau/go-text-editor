package main

// ✅ (1) Lineable (L)과 Flatted (F) 타입 제약 정의
type Lineable interface {
	LinePieceTable | LineScreenBuffer
}

type Flatted interface {
	~[]byte | ~[]uint32
}

// ✅ (2) LinePieceTable과 LineScreenBuffer 정의
type LinePieceTable struct {
	data []byte
}

func (l LinePieceTable) Flatten() []byte {
	return l.data
}

func (l LinePieceTable) Data() LinePieceTable {
	return l
}

type LineScreenBuffer struct {
	data []uint32
}

func (l LineScreenBuffer) Flatten() []uint32 {
	return l.data
}

func (l LineScreenBuffer) Data() LineScreenBuffer {
	return l
}

// ✅ (3) 제너릭 인터페이스: Line (L: Lineable, F: Flatted)
type Line[L Lineable, F Flatted] interface {
	Flatten() F
	Data() L
}

// ✅ (4) 제너릭 구조체: Lines (L: Lineable, F: Flatted)
type Lines[L Lineable, F Flatted] struct {
	tree []Line[L, F] // ✅ `L`과 `F`를 반드시 명시해야 함
}

// ✅ (5) InsertLine 메서드
func (l *Lines[L, F]) InsertLine(line Line[L, F]) {
	l.tree = append(l.tree, line)
}

// ✅ (6) 예제 실행 코드
func main() {
	// ✅ LinePieceTable을 사용하는 Lines 생성
	lines := Lines[LinePieceTable, []byte]{}
	lines.InsertLine(LinePieceTable{data: []byte("Hello, World!")})

	// ✅ LineScreenBuffer를 사용하는 Lines 생성
	screenLines := Lines[LineScreenBuffer, []uint32]{}
	screenLines.InsertLine(LineScreenBuffer{data: []uint32{1, 2, 3, 4}})

	println("Lines inserted successfully")
}
