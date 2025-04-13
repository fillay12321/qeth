@echo off
echo Запуск установки QETH для Windows...

REM Проверка наличия PowerShell
where powershell >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo Ошибка: PowerShell не найден. Установите Windows PowerShell или PowerShell Core.
    exit /b 1
)

REM Получаем путь к скрипту
set "SCRIPT_PATH=%~dp0setup.ps1"

REM Запускаем PowerShell скрипт с повышенными правами
powershell -ExecutionPolicy Bypass -File "%SCRIPT_PATH%"

if %ERRORLEVEL% neq 0 (
    echo Ошибка: Не удалось выполнить PowerShell скрипт.
    exit /b 1
)

echo Установка QETH успешно завершена!
pause 