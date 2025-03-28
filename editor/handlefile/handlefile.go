package handlefile

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/joho/godotenv" // godotenv 라이브러리 사용
)

// ✅ `init()`을 사용해 자동 실행
func init() {
	LoadEnv()
}

// ✅ Git 루트 디렉토리 찾기
func GetProjectRoot() string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Git 루트 디렉토리를 찾을 수 없습니다: %v", err)
		// 현재 작업 디렉토리를 대체로 사용
		currentDir, err := os.Getwd()
		if err != nil {
			log.Fatal("현재 디렉토리를 확인할 수 없습니다.")
		}
		log.Printf("현재 작업 디렉토리를 프로젝트 루트로 사용합니다: %s", currentDir)
		return currentDir
	}
	return string(output[:len(output)-1]) // 개행 문자 제거
}

// ✅ 환경변수 로드 함수
func LoadEnv() {
	projectRoot := GetProjectRoot()
	envLocalPath := filepath.Join(projectRoot, ".env.local")
	envPath := filepath.Join(projectRoot, ".env")
	envExamplePath := filepath.Join(projectRoot, ".env.example")

	// 우선순위: .env.local > .env > .env.example
	if _, err := os.Stat(envLocalPath); err == nil {
		err = godotenv.Load(envLocalPath)
		if err != nil {
			log.Printf("⚠️ .env.local 파일 로드 오류: %v", err)
		} else {
			log.Println("✅ .env.local 로드 완료")
			return
		}
	}

	if _, err := os.Stat(envPath); err == nil {
		err = godotenv.Load(envPath)
		if err != nil {
			log.Printf("⚠️ .env 파일 로드 오류: %v", err)
		} else {
			log.Println("✅ .env 로드 완료")
			return
		}
	}

	if _, err := os.Stat(envExamplePath); err == nil {
		err = godotenv.Load(envExamplePath)
		if err != nil {
			log.Printf("⚠️ .env.example 파일 로드 오류: %v", err)
		} else {
			log.Println("✅ .env.example 로드 완료")
			return
		}
	}

	// 환경변수 파일이 없는 경우, 기본 파일 생성
	log.Println("⚠️ 환경변수 파일을 찾을 수 없습니다. 기본 .env 파일을 생성합니다.")
	createDefaultEnvFile(envPath)
}

// ✅ 기본 환경변수 파일 생성
func createDefaultEnvFile(filepath string) {
	content := `# 에디터 환경 설정 파일
# 이 파일은 에디터의 환경 설정을 관리합니다.

# 저장 파일 경로 설정 (절대 경로 권장)
SAVE_TXT="/home/rlaaudgjs5638/go_editor/saved.txt"

# 기타 설정을 추가할 수 있습니다.
# THEME="dark"
# FONT_SIZE="12"
`

	err := os.WriteFile(filepath, []byte(content), 0644)
	if err != nil {
		log.Printf("⚠️ 기본 .env 파일 생성 실패: %v", err)
		return
	}

	err = godotenv.Load(filepath)
	if err != nil {
		log.Printf("⚠️ 생성된 .env 파일 로드 실패: %v", err)
		return
	}

	log.Println("✅ 기본 .env 파일 생성 및 로드 완료")
}

// ✅ SAVE_TXT 환경변수에서 저장 파일 경로를 가져옴
func GetSaveTxtPath() string {
	// 환경변수에서 SAVE_TXT 값을 읽어옴 (없으면 기본값 사용)
	filePath := os.Getenv("SAVE_TXT")
	if filePath == "" {
		projectRoot := GetProjectRoot()
		filePath = filepath.Join(projectRoot, "saved.txt")
		log.Printf("⚠️ SAVE_TXT 환경변수가 설정되지 않았습니다. 기본값을 사용합니다: %s", filePath)
	}
	return filePath
}
