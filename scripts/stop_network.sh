#!/bin/bash

# Скрипт для остановки локальной сети узлов AetherWave

# Цвета для вывода
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Директория с PID процессов
PID_DIR="./.pid"

# Проверяем существование директории
if [ ! -d "$PID_DIR" ]; then
    echo -e "${RED}Директория с PID не найдена. Возможно, узлы не запущены.${NC}"
    exit 1
fi

# Ищем все файлы PID
PID_FILES=$(find $PID_DIR -name "*.pid")

if [ -z "$PID_FILES" ]; then
    echo -e "${YELLOW}PID файлы не найдены. Возможно, узлы не запущены.${NC}"
    exit 0
fi

# Останавливаем все процессы
echo -e "${YELLOW}Останавливаем узлы сети...${NC}"

for pid_file in $PID_FILES; do
    if [ -f "$pid_file" ]; then
        node_name=$(basename "$pid_file" .pid)
        pid=$(cat "$pid_file")
        
        if ps -p $pid > /dev/null; then
            echo -e "${YELLOW}Останавливаем $node_name (PID: $pid)...${NC}"
            kill $pid
            # Ждем завершения процесса
            for i in {1..5}; do
                if ! ps -p $pid > /dev/null; then
                    break
                fi
                sleep 1
            done
            
            # Если процесс все еще работает, завершаем его принудительно
            if ps -p $pid > /dev/null; then
                echo -e "${RED}Процесс $node_name не завершился, принудительное завершение...${NC}"
                kill -9 $pid
            fi
        else
            echo -e "${YELLOW}Процесс $node_name (PID: $pid) уже не работает${NC}"
        fi
        
        # Удаляем файл PID
        rm "$pid_file"
    fi
done

echo -e "${GREEN}Все узлы сети остановлены.${NC}" 