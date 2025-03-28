package syncer

import (
	"fmt"
	"go_editor/editor/handlefile"
	glp "go_editor/editor/screener/glyph"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// ----------------------------------------------------
// (3) SyncProtocol 생성자 with file loading
// ----------------------------------------------------
func LoadSyncProtocol(screenWidth, screenHeight int, fg, bg uint32, LineHeight int) *SyncProtocol {
	// 환경변수에서 SAVE_TXT 경로 가져오기
	filePath := handlefile.GetSaveTxtPath()
	log.Printf("Using SAVE_TXT path: %s", filePath)

	lineCount := screenHeight / LineHeight
	syncData := &SyncData{}

	// 파일이 존재하면 파일 내용 로드
	var lines []string
	fileData, err := os.ReadFile(filePath)
	if err == nil {
		// 파일 내용을 라인별로 분리
		fileLines := strings.Split(string(fileData), "\n")
		// Windows 파일에서 \r\n 처리를 위해 \r 제거
		for _, line := range fileLines {
			lines = append(lines, strings.TrimRight(line, "\r"))
		}
		log.Printf("✅ %d 라인을 로드했습니다. 파일: %s", len(lines), filePath)
	} else {
		// 파일이 없는 경우 빈 문서로 처리
		if os.IsNotExist(err) {
			log.Printf("⚠️ 파일이 존재하지 않습니다. 빈 문서로 시작합니다.")
		} else {
			log.Printf("⚠️ 파일 로드 오류: %v", err)
		}
	}

	// 파일 내용으로 노드 생성
	var curNode *SyncNode = nil
	for i := range lineCount {
		if i < len(lines) {
			// 파일에서 읽은 라인으로 노드 추가
			curNode = syncData.appendByPtr(curNode, lines[i])
		} else {

			// 파일 라인이 부족하면 빈 라인 추가
			curNode = syncData.appendByPtr(curNode, "")

		}
	}
	PrintList2(syncData.head)
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
	log.Printf("✅ %d 라인 초기화 완료", lineCount)
	PrintList2(sp.syncData.head)
	return sp
}

// PrintList: head부터 next 방향으로 순회하며 출력
func PrintList2(head *SyncNode) {
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
		f := false
		if cur.LineBuffer == nil {
			f = true
		}
		fmt.Printf(" Node[%d]: %q void: %v\n", i, cur.PieceTable.String(), f)
		cur = cur.next
		i++
	}
}

// SaveToFile 환경변수에 정의된 위치에 문서 저장
func (sp *SyncProtocol) SaveToFile() error {
	filePath := handlefile.GetSaveTxtPath()

	// 내용을 파일로 저장하기 위한 텍스트 수집
	var lines []string

	sp.syncData.ForEach(func(sn *SyncNode) {
		// PieceTable에서 문자열 가져오기
		lineText := sn.PieceTable.String()
		lines = append(lines, lineText)
	})

	// 마지막의 빈 라인들은 저장하지 않음 (실제 문서 내용만 저장)
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i] != "" {
			lines = lines[:i+1]
			break
		}
		// 모든 라인이 비어있으면 빈 파일 저장
		if i == 0 {
			lines = []string{""}
		}
	}

	// 저장 디렉토리가 존재하는지 확인하고 없으면 생성
	dir := filepath.Dir(filePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("디렉토리 생성 실패: %v", err)
		}
	}

	// 내용을 파일로 저장
	content := strings.Join(lines, "\n")
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return err
	}

	log.Printf("✅ %d 라인을 파일에 저장했습니다: %s", len(lines), filePath)
	return nil
}
