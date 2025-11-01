package logging

import (
	"log"
	"os"
	"path/filepath"
)

func InitLogger(logFilePath string) {

	dir := filepath.Dir(logFilePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Не удалось создать директорию логов: %v", err)
		}
	}

	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Не удалось открыть лог-файл: %v", err)
	}

	log.SetOutput(file)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("=====> Start logging....")
}
