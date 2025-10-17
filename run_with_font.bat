@echo off
echo ====================================
echo 打印工具 - 桌面版（中文字体支持）
echo ====================================
echo.

REM 设置中文字体环境变量
set FYNE_FONT=PingFang Regular_0.ttf

echo 正在启动（使用中文字体）...
go run main_desktop.go print_functions.go

pause
