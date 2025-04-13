# PowerShell скрипт для настройки Quest на Windows

# Функция для вывода сообщений с цветом
function Write-ColorLog {
    param ([string]$Message)
    Write-Host "[QUEST-SETUP] $Message" -ForegroundColor Blue
}

# Основная функция
function Main {
    Write-ColorLog "Настройка QuEST начата..."
    
    # Проверяем зависимости
    Write-ColorLog "Проверка зависимостей..."
    
    # Проверка Git
    if (!(Get-Command git -ErrorAction SilentlyContinue)) {
        Write-Host "Ошибка: git не установлен. Пожалуйста, установите git: https://git-scm.com/download/win" -ForegroundColor Red
        exit 1
    }
    
    # Проверка CMake
    if (!(Get-Command cmake -ErrorAction SilentlyContinue)) {
        Write-Host "CMake не найден. Пожалуйста, установите CMake: https://cmake.org/download/" -ForegroundColor Yellow
        Write-Host "После установки CMake перезапустите скрипт." -ForegroundColor Yellow
        exit 1
    }
    
    # Скачиваем QuEST
    $QUEST_KIT_REPO = "https://github.com/QuEST-Kit/QuEST.git"
    $QUEST_KIT_DIR = Join-Path $PSScriptRoot "..\..\quest-kit"
    
    Write-ColorLog "Скачиваем QuEST..."
    
    if (Test-Path $QUEST_KIT_DIR) {
        Write-ColorLog "QuEST уже скачан. Обновляем..."
        Push-Location $QUEST_KIT_DIR
        git pull
        Pop-Location
    } else {
        Write-ColorLog "Скачиваем QuEST..."
        git clone --depth 1 $QUEST_KIT_REPO $QUEST_KIT_DIR
    }
    
    # Компиляция QuEST
    Write-ColorLog "Компилируем QuEST..."
    
    # Создаем директорию для сборки
    $BUILD_DIR = Join-Path $QUEST_KIT_DIR "build"
    if (!(Test-Path $BUILD_DIR)) {
        New-Item -ItemType Directory -Path $BUILD_DIR | Out-Null
    }
    
    # Собираем с помощью CMake
    Push-Location $BUILD_DIR
    
    # Используем подходящий генератор - автоопределение
    cmake ..
    cmake --build . --config Release
    
    Pop-Location
    
    # Настройка Go-привязки
    Write-ColorLog "Настраиваем Go-привязку для QuEST..."
    
    # Создаем директории для заголовочных файлов и библиотек
    $QUEST_DIR = Join-Path $PSScriptRoot "..\..\quest"
