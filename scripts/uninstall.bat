@echo off
setlocal enabledelayedexpansion

:: Check for administrative privileges
net session >nul 2>&1
if %errorLevel% == 0 (
    goto :admin_ok
) else (
    echo Administrator privileges are required to uninstall the context menu item.
    echo Please right-click this script and select "Run as administrator".
    pause
    exit /b 1
)

:admin_ok

echo Removing context menu entries from registry...

:: Remove Folder context menu
reg delete "HKCR\Directory\shell\Recolor" /f >nul 2>&1

:: Remove image files context menu
reg delete "HKCR\SystemFileAssociations\.png\shell\Recolor" /f >nul 2>&1
reg delete "HKCR\SystemFileAssociations\.jpg\shell\Recolor" /f >nul 2>&1
reg delete "HKCR\SystemFileAssociations\.jpeg\shell\Recolor" /f >nul 2>&1

echo Removing application files...
set "APP_DIR=%ProgramFiles%\recol"
if exist "%APP_DIR%" rmdir /s /q "%APP_DIR%"

echo.
echo Uninstallation complete!
pause
