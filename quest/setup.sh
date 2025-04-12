#!/bin/bash
set -e

QUEST_KIT_VERSION="4.0.0"
QUEST_KIT_REPO="https://github.com/QuEST-Kit/QuEST.git"
QUEST_KIT_DIR="$(pwd)/quest-kit"

# Создаем цветную функцию для логов
log() {
    echo -e "\033[0;34m[QUEST-SETUP]\033[0m $1"
}

# Проверяем наличие git и cmake
check_dependencies() {
    log "Проверка зависимостей..."
    if ! command -v git &> /dev/null; then
        echo "Ошибка: git не установлен. Установите git и попробуйте снова."
        exit 1
    fi
    
    if ! command -v cmake &> /dev/null; then
        echo "Ошибка: cmake не установлен. Установите cmake и попробуйте снова."
        exit 1
    fi
    
    if ! command -v make &> /dev/null; then
        echo "Ошибка: make не установлен. Установите make и попробуйте снова."
        exit 1
    fi
}

# Скачиваем quest-kit если его нет
download_quest_kit() {
    if [ -d "$QUEST_KIT_DIR" ]; then
        log "QuEST уже скачан. Обновляем..."
        cd "$QUEST_KIT_DIR"
        git pull
        cd -
    else
        log "Скачиваем QuEST версии $QUEST_KIT_VERSION..."
        git clone --depth 1 "$QUEST_KIT_REPO" "$QUEST_KIT_DIR"
    fi
}

# Компилируем quest-kit
build_quest_kit() {
    log "Компилируем QuEST..."
    mkdir -p "$QUEST_KIT_DIR/build"
    cd "$QUEST_KIT_DIR/build"
    cmake ..
    make -j4
    cd -
    log "QuEST успешно скомпилирован!"
}

# Копируем необходимые файлы для Go-привязки
setup_go_binding() {
    log "Настраиваем Go-привязку для QuEST..."
    mkdir -p quest/include
    mkdir -p quest/lib
    
    # Копируем заголовочные файлы
    cp -r "$QUEST_KIT_DIR/quest/include/"* quest/include/ 2>/dev/null || cp -r "$QUEST_KIT_DIR/include/"* quest/include/ 2>/dev/null || :
    
    # Копируем библиотеки
    cp "$QUEST_KIT_DIR/build/quest/libQuEST.a" quest/lib/libquestkit.a 2>/dev/null || cp "$QUEST_KIT_DIR/build/libQuEST.a" quest/lib/libquestkit.a 2>/dev/null || :
    cp "$QUEST_KIT_DIR/build/quest/libQuEST.so" quest/lib/libquestkit.so 2>/dev/null || cp "$QUEST_KIT_DIR/build/libQuEST.so" quest/lib/libquestkit.so 2>/dev/null || :
    
    log "Go-привязка настроена!"
}

main() {
    log "Настройка QuEST начата..."
    check_dependencies
    download_quest_kit
    build_quest_kit
    setup_go_binding
    log "Настройка QuEST завершена успешно!"
}

main 