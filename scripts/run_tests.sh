#!/bin/bash

# Скрипт для запуска тестов с расширенными опциями и логированием ошибок

# Создаем директории для логов и отчетов о покрытии
mkdir -p test-logs coverage

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Функция для запуска тестов с определенными параметрами
run_tests() {
    local test_path=$1
    local log_file=$2
    local coverage_file=$3
    local coverage_html=$4
    local test_name=$5

    echo -e "${YELLOW}Running $test_name tests...${NC}"
    
    # Запускаем тесты с покрытием и записываем вывод в лог
    go test $test_path -v -coverprofile=$coverage_file -covermode=atomic -timeout=30s 2>&1 | tee $log_file
    
    # Проверяем результат выполнения тестов
    if [ ${PIPESTATUS[0]} -eq 0 ]; then
        echo -e "${GREEN}$test_name tests passed successfully!${NC}"
    else
        echo -e "${RED}$test_name tests failed! See $log_file for details.${NC}"
        echo "Test failures detected. Check $log_file for details." >> test-logs/failures.log
        echo "Failed test: $test_name" >> test-logs/failures.log
        echo "Timestamp: $(date)" >> test-logs/failures.log
        echo "----------------------------------------" >> test-logs/failures.log
    fi
    
    # Генерируем HTML-отчет о покрытии
    go tool cover -html=$coverage_file -o $coverage_html
    
    echo -e "${YELLOW}Coverage report generated at $coverage_html${NC}"
    echo
}

# Очищаем файл с ошибками
> test-logs/failures.log

# Запускаем модульные тесты
run_tests "./pkg/..." "test-logs/unit_tests.log" "coverage/unit_coverage.out" "coverage/unit_coverage.html" "Unit"

# Запускаем интеграционные тесты
run_tests "./tests/..." "test-logs/integration_tests.log" "coverage/integration_coverage.out" "coverage/integration_coverage.html" "Integration"

# Запускаем все тесты вместе для общего покрытия
run_tests "./..." "test-logs/all_tests.log" "coverage/coverage.out" "coverage/coverage.html" "All"

# Выводим общую статистику покрытия
echo -e "${YELLOW}Overall test coverage:${NC}"
go tool cover -func=coverage/coverage.out

# Проверяем, были ли ошибки
if [ -s test-logs/failures.log ]; then
    echo -e "${RED}Some tests failed! See test-logs/failures.log for details.${NC}"
    exit 1
else
    echo -e "${GREEN}All tests passed successfully!${NC}"
    exit 0
fi 