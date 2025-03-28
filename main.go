package main

import (
	"go_editor/editor"
)

func main() {

	// Editor 생성 (800x600, 30FPS)
	edt, err := editor.NewEditor(800, 600, 30)
	if err != nil {
		panic(err)
	}

	// 메인 이벤트 루프 실행
	edt.Run()

	// 종료 후 정리
}
