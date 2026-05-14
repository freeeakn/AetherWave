//go:build ignore

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"time"

	"github.com/freeeakn/AetherWave/pkg/blockchain"
	"github.com/freeeakn/AetherWave/pkg/crypto"
)

// Типы профилирования
const (
	ProfileCPU    = "cpu"
	ProfileMemory = "memory"
	ProfileBlock  = "block"
)

func main() {
	// Флаги командной строки
	profileType := flag.String("type", "cpu", "Тип профилирования: cpu, memory, block")
	messageCount := flag.Int("messages", 100, "Количество сообщений для теста")
	outputFile := flag.String("output", "", "Файл для сохранения результатов профилирования")
	duration := flag.Duration("duration", 30*time.Second, "Продолжительность профилирования")
	difficulty := flag.Int("difficulty", 3, "Сложность майнинга")
	flag.Parse()

	// Проверяем и создаем директорию для профилей
	profileDir := "profiles"
	if _, err := os.Stat(profileDir); os.IsNotExist(err) {
		os.Mkdir(profileDir, 0755)
	}

	// Если выходной файл не указан, используем значение по умолчанию
	if *outputFile == "" {
		timestamp := time.Now().Format("20060102_150405")
		*outputFile = fmt.Sprintf("%s/%s_%s.prof", profileDir, *profileType, timestamp)
	}

	fmt.Printf("Запуск профилирования типа '%s', результаты будут сохранены в '%s'\n", *profileType, *outputFile)

	// Запускаем соответствующий тип профилирования
	switch *profileType {
	case ProfileCPU:
		profileCPU(*outputFile, *messageCount, *duration, *difficulty)
	case ProfileMemory:
		profileMemory(*outputFile, *messageCount, *duration, *difficulty)
	case ProfileBlock:
		profileBlockCreation(*outputFile, *messageCount, *difficulty)
	default:
		log.Fatalf("Неизвестный тип профилирования: %s", *profileType)
	}
}

// profileCPU выполняет профилирование CPU
func profileCPU(outputFile string, messageCount int, duration time.Duration, difficulty int) {
	// Создаем файл для профиля
	f, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("Не удалось создать файл профиля CPU: %v", err)
	}
	defer f.Close()

	// Запускаем профилирование CPU
	if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatalf("Не удалось запустить профилирование CPU: %v", err)
	}
	defer pprof.StopCPUProfile()

	fmt.Printf("Профилирование CPU запущено на %v...\n", duration)

	// Создаем блокчейн с заданной сложностью
	bc := blockchain.NewBlockchain()
	bc.Difficulty = difficulty

	// Генерируем ключ для шифрования
	key, err := crypto.GenerateKey()
	if err != nil {
		log.Fatalf("Ошибка генерации ключа: %v", err)
	}

	// Запускаем тестовую нагрузку
	deadline := time.Now().Add(duration)
	messagesAdded := 0

	for time.Now().Before(deadline) && messagesAdded < messageCount {
		// Добавляем сообщение
		err := bc.AddMessage(
			fmt.Sprintf("Sender-%d", messagesAdded),
			fmt.Sprintf("Recipient-%d", messagesAdded),
			fmt.Sprintf("Message content %d", messagesAdded),
			key,
		)
		if err != nil {
			fmt.Printf("Ошибка при добавлении сообщения: %v\n", err)
			break
		}
		messagesAdded++

		// Выводим прогресс
		if messagesAdded%10 == 0 {
			fmt.Printf("Добавлено %d/%d сообщений\n", messagesAdded, messageCount)
		}
	}

	fmt.Printf("Профилирование CPU завершено. Добавлено %d сообщений.\n", messagesAdded)
	fmt.Printf("Для анализа результатов используйте: go tool pprof %s\n", outputFile)
}

// profileMemory выполняет профилирование использования памяти
func profileMemory(outputFile string, messageCount int, duration time.Duration, difficulty int) {
	// Создаем блокчейн с заданной сложностью
	bc := blockchain.NewBlockchain()
	bc.Difficulty = difficulty

	// Генерируем ключ для шифрования
	key, err := crypto.GenerateKey()
	if err != nil {
		log.Fatalf("Ошибка генерации ключа: %v", err)
	}

	// Запускаем тестовую нагрузку
	deadline := time.Now().Add(duration)
	messagesAdded := 0

	for time.Now().Before(deadline) && messagesAdded < messageCount {
		// Добавляем сообщение
		err := bc.AddMessage(
			fmt.Sprintf("Sender-%d", messagesAdded),
			fmt.Sprintf("Recipient-%d", messagesAdded),
			fmt.Sprintf("Message content %d", messagesAdded),
			key,
		)
		if err != nil {
			fmt.Printf("Ошибка при добавлении сообщения: %v\n", err)
			break
		}
		messagesAdded++

		// Выводим прогресс
		if messagesAdded%10 == 0 {
			fmt.Printf("Добавлено %d/%d сообщений\n", messagesAdded, messageCount)
		}
	}

	// Сохраняем профиль памяти
	f, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("Не удалось создать файл профиля памяти: %v", err)
	}
	defer f.Close()

	if err := pprof.WriteHeapProfile(f); err != nil {
		log.Fatalf("Не удалось записать профиль памяти: %v", err)
	}

	fmt.Printf("Профилирование памяти завершено. Добавлено %d сообщений.\n", messagesAdded)
	fmt.Printf("Для анализа результатов используйте: go tool pprof %s\n", outputFile)
}

// profileBlockCreation выполняет профилирование создания блоков
func profileBlockCreation(outputFile string, blockCount int, difficulty int) {
	// Создаем блокчейн с заданной сложностью
	bc := blockchain.NewBlockchain()
	bc.Difficulty = difficulty

	// Генерируем ключ для шифрования
	key, err := crypto.GenerateKey()
	if err != nil {
		log.Fatalf("Ошибка генерации ключа: %v", err)
	}

	// Измеряем время создания каждого блока
	results := make([]time.Duration, 0, blockCount)
	totalMessages := 0

	fmt.Printf("Запуск профилирования создания %d блоков со сложностью %d...\n", blockCount, difficulty)

	for i := 0; i < blockCount; i++ {
		// Добавляем случайное количество сообщений (1-5)
		messagesCount := (i % 5) + 1
		totalMessages += messagesCount

		startTime := time.Now()

		for j := 0; j < messagesCount; j++ {
			// Добавляем сообщение
			err := bc.AddMessage(
				fmt.Sprintf("Sender-%d-%d", i, j),
				fmt.Sprintf("Recipient-%d-%d", i, j),
				fmt.Sprintf("Message content %d-%d", i, j),
				key,
			)
			if err != nil {
				fmt.Printf("Ошибка при добавлении сообщения: %v\n", err)
				break
			}
		}

		duration := time.Since(startTime)
		results = append(results, duration)

		fmt.Printf("Блок %d/%d создан за %v (сообщений: %d)\n", i+1, blockCount, duration, messagesCount)
	}

	// Вычисляем статистику
	var totalDuration time.Duration
	var minDuration time.Duration = results[0]
	var maxDuration time.Duration = results[0]

	for _, duration := range results {
		totalDuration += duration
		if duration < minDuration {
			minDuration = duration
		}
		if duration > maxDuration {
			maxDuration = duration
		}
	}

	averageDuration := totalDuration / time.Duration(len(results))

	// Записываем результаты в файл
	f, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("Не удалось создать файл результатов: %v", err)
	}
	defer f.Close()

	fmt.Fprintf(f, "Block Creation Profile\n")
	fmt.Fprintf(f, "Difficulty: %d\n", difficulty)
	fmt.Fprintf(f, "Total blocks: %d\n", blockCount)
	fmt.Fprintf(f, "Total messages: %d\n", totalMessages)
	fmt.Fprintf(f, "Average creation time: %v\n", averageDuration)
	fmt.Fprintf(f, "Min creation time: %v\n", minDuration)
	fmt.Fprintf(f, "Max creation time: %v\n", maxDuration)
	fmt.Fprintf(f, "\nDetailed Results:\n")

	for i, duration := range results {
		fmt.Fprintf(f, "Block %d: %v\n", i+1, duration)
	}

	fmt.Printf("\nРезультаты профилирования создания блоков:\n")
	fmt.Printf("Всего блоков: %d\n", blockCount)
	fmt.Printf("Всего сообщений: %d\n", totalMessages)
	fmt.Printf("Среднее время создания: %v\n", averageDuration)
	fmt.Printf("Минимальное время: %v\n", minDuration)
	fmt.Printf("Максимальное время: %v\n", maxDuration)
	fmt.Printf("Подробные результаты сохранены в файле: %s\n", outputFile)
}
