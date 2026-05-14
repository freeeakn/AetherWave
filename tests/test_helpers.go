package tests

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestLogger представляет собой логгер для тестов
type TestLogger struct {
	t      *testing.T
	prefix string
	file   *os.File
}

// NewTestLogger создает новый логгер для тестов
func NewTestLogger(t *testing.T, prefix string) *TestLogger {
	// Создаем директорию для логов, если она не существует
	logDir := "test-logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("Failed to create log directory: %v", err)
	}

	// Создаем файл для логов
	logFile := filepath.Join(logDir, fmt.Sprintf("%s.log", t.Name()))
	file, err := os.Create(logFile)
	if err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}

	return &TestLogger{
		t:      t,
		prefix: prefix,
		file:   file,
	}
}

// Log записывает сообщение в лог
func (l *TestLogger) Log(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	// Получаем информацию о вызывающей функции
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "unknown"
		line = 0
	}

	// Сокращаем путь к файлу
	file = filepath.Base(file)

	// Форматируем сообщение
	logMsg := fmt.Sprintf("[%s] [%s] [%s:%d] %s\n", timestamp, l.prefix, file, line, msg)

	// Записываем в файл
	if _, err := l.file.WriteString(logMsg); err != nil {
		l.t.Errorf("Failed to write to log file: %v", err)
	}

	// Выводим в консоль
	l.t.Logf("%s: %s", l.prefix, msg)
}

// Close закрывает файл лога
func (l *TestLogger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}

// SetupTestLogging настраивает логирование для тестов
func SetupTestLogging(t *testing.T) (*TestLogger, func()) {
	logger := NewTestLogger(t, t.Name())

	// Возвращаем функцию для очистки
	cleanup := func() {
		logger.Close()
	}

	return logger, cleanup
}

// CaptureOutput перехватывает вывод stdout и stderr
func CaptureOutput(f func()) string {
	// Сохраняем оригинальные stdout и stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	// Создаем пайпы для перехвата вывода
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	// Запускаем функцию
	f()

	// Закрываем writer, чтобы все данные были записаны
	w.Close()

	// Читаем данные из reader
	var buf strings.Builder
	io.Copy(&buf, r)

	// Восстанавливаем оригинальные stdout и stderr
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	return buf.String()
}

// AssertNoError проверяет, что ошибка равна nil
func AssertNoError(t *testing.T, err error, msg string) {
	if err != nil {
		t.Errorf("%s: %v", msg, err)
	}
}

// AssertError проверяет, что ошибка не равна nil
func AssertError(t *testing.T, err error, msg string) {
	if err == nil {
		t.Errorf("%s: expected error, got nil", msg)
	}
}

// AssertEqual проверяет, что два значения равны
func AssertEqual(t *testing.T, expected, actual interface{}, msg string) {
	if expected != actual {
		t.Errorf("%s: expected %v, got %v", msg, expected, actual)
	}
}

// AssertTrue проверяет, что условие истинно
func AssertTrue(t *testing.T, condition bool, msg string) {
	if !condition {
		t.Errorf("%s: expected true, got false", msg)
	}
}

// AssertFalse проверяет, что условие ложно
func AssertFalse(t *testing.T, condition bool, msg string) {
	if condition {
		t.Errorf("%s: expected false, got true", msg)
	}
}

// LogTestInfo записывает информацию о тесте в лог-файл
func LogTestInfo(testName string, info string) {
	// Создаем директорию для логов, если она не существует
	logDir := "test-logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("Failed to create log directory: %v", err)
		return
	}

	// Открываем файл для логов
	logFile := filepath.Join(logDir, "test_info.log")
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open log file: %v", err)
		return
	}
	defer file.Close()

	// Записываем информацию
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	logMsg := fmt.Sprintf("[%s] [%s] %s\n", timestamp, testName, info)
	if _, err := file.WriteString(logMsg); err != nil {
		log.Printf("Failed to write to log file: %v", err)
	}
}
