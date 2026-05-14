#!/bin/bash

# Цвета для вывода
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Запуск тестов для проекта AetherWave${NC}"
echo "======================================"

# Функция для запуска тестов в указанном пакете
run_tests() {
    local package=$1
    local package_name=$2
    
    echo -e "${YELLOW}Запуск тестов для пакета $package_name${NC}"
    echo "--------------------------------------"
    
    # Запускаем тесты с подробным выводом и покрытием кода
    go test -v -cover $package
    
    # Проверяем код возврата
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Тесты для пакета $package_name успешно пройдены${NC}"
    else
        echo -e "${RED}Тесты для пакета $package_name завершились с ошибками${NC}"
        FAILED=1
    fi
    echo ""
}

# Переменная для отслеживания ошибок
FAILED=0

# Запускаем тесты для каждого пакета
run_tests "./pkg/crypto" "crypto"
run_tests "./pkg/blockchain" "blockchain"
run_tests "./pkg/network" "network"
run_tests "./tests" "integration tests"

# Выводим общий результат
echo "======================================"
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}Все тесты успешно пройдены!${NC}"
else
    echo -e "${RED}Некоторые тесты завершились с ошибками.${NC}"
    exit 1
fi 