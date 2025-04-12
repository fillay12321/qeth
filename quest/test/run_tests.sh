#!/bin/bash

# Скрипт для запуска тестов интеграции Quest с EVM

# Переходим в корневую директорию проекта
cd "$(dirname "$0")/../.." || exit 1

# Проверяем, что quest-kit установлен
if [ ! -d "quest/quest-kit" ]; then
    echo "Ошибка: quest-kit не установлен. Запустите setup.sh сначала."
    exit 1
fi

# Компилируем go-ethereum с поддержкой Quest
echo "Компиляция go-ethereum с поддержкой Quest..."
make geth || exit 1

# Запускаем тесты Quest
echo "Запуск тестов интеграции Quest..."
go test -v ./quest/test

# Запускаем бенчмарки
echo "Запуск бенчмарков Quest vs EVM..."
go test -bench=. ./quest/test 