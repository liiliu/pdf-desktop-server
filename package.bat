@echo off
echo ====================================
echo Package Script
echo ====================================
echo.

set RELEASE_DIR=PrintToolRelease

if exist "%RELEASE_DIR%" (
    echo Cleaning...
    rmdir /s /q "%RELEASE_DIR%"
)

mkdir "%RELEASE_DIR%"
mkdir "%RELEASE_DIR%\images"
mkdir "%RELEASE_DIR%\pdfs"
mkdir "%RELEASE_DIR%\resources"
mkdir "%RELEASE_DIR%\resources\fonts"
mkdir "%RELEASE_DIR%\resources\images"

echo.
echo Compiling...
go build -ldflags="-H windowsgui" -o "%RELEASE_DIR%\PrintTool.exe" main_desktop.go print_functions.go

if %ERRORLEVEL% NEQ 0 (
    echo Build failed!
    pause
    exit /b 1
)

echo.
echo Copying...

copy config.toml "%RELEASE_DIR%\" >nul
copy resources\images\*-69.png "%RELEASE_DIR%\resources\images\" >nul
copy resources\images\favicon.ico "%RELEASE_DIR%\resources\images\" >nul
copy resources\fonts\PingFang*.ttf "%RELEASE_DIR%\resources\fonts\" >nul

echo Done copying basic files

powershell -Command "Copy-Item '*.txt' '%RELEASE_DIR%\'"

echo.
echo ====================================
echo Completed!
echo ====================================
dir "%RELEASE_DIR%" /b
echo.
pause
