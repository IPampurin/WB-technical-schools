package server

import (
	"fmt"
	"log"
	"os"
	"time"
)

// SetupLogging создает и настраивает логгер с записью в файл,
// возвращает логгер и файл для закрытия или ошибку
func SetupLogging() (*log.Logger, *os.File, error) {

	// создаем папку logs если нет (владелец читает/пишет/исполняет, остальные читают/исполняют)
	if err := os.MkdirAll("logs", 0755); err != nil {
		return nil, nil, fmt.Errorf("не удалось создать папку logs: %w", err)
	}

	// генерируем имя файла с текущей датой
	filename := fmt.Sprintf("logs/calendar_%s.log", time.Now().Format("2006-01-02"))

	// открываем файл для записи:
	// - os.O_CREATE: создать если не существует
	// - os.O_WRONLY: только запись
	// - os.O_APPEND: дописывать в конец (не перезаписывать)
	// - 0666: права чтения/записи для всех
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, nil, fmt.Errorf("не удалось открыть файл логов: %w", err)
	}
	// закрываем файл из server.Run

	// создаем логгер с настройками:
	// - file: куда писать логи
	// - "": префикс пустой
	// - log.LstdFlags == 0 (флаги даты и времени (2009/01/23 01:23:23)) у нас и так пишутся
	logger := log.New(file, "", 0)

	return logger, file, nil
}
