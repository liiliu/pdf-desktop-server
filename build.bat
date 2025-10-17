@echo off
echo ====================================
echo 打印工具 - 桌面版编译脚本
echo ====================================
echo.

echo 正在编译桌面应用...
go build -o print_desktop.exe main_desktop.go print_functions.go

if %ERRORLEVEL% EQU 0 (
    echo.
    echo ====================================
    echo 编译成功！
    echo 可执行文件: print_desktop.exe
    echo ====================================
    echo.
    echo 运行程序:
    echo   print_desktop.exe
    echo.
) else (
    echo.
    echo ====================================
    echo 编译失败！请检查错误信息
    echo ====================================
    echo.
)

pause
