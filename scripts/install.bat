@echo off
setlocal enabledelayedexpansion

:: Check for administrative privileges
net session >nul 2>&1
if %errorLevel% == 0 (
    goto :admin_ok
) else (
    echo Administrator privileges are required to install the context menu item.
    echo Please right-click this script and select "Run as administrator".
    pause
    exit /b 1
)

:admin_ok

:: Set paths
set "APP_DIR=%ProgramFiles%\recol"
set "EXE_PATH=%APP_DIR%\recol.exe"

:: Create directory and copy executable
echo Installing recol to %APP_DIR%...
if not exist "%APP_DIR%" mkdir "%APP_DIR%"
copy /y "%~dp0recol.exe" "%EXE_PATH%"
if %errorLevel% neq 0 (
    echo Failed to copy recol.exe to %APP_DIR%.
    echo Ensure recol.exe is in the same directory as this script.
    pause
    exit /b 1
)

echo Adding context menu entries to registry...

:: Add to Folder context menu
reg add "HKCR\Directory\shell\Recolor" /v "" /t REG_SZ /d "Recolor (recol)" /f
reg add "HKCR\Directory\shell\Recolor" /v "Icon" /t REG_SZ /d "\"%EXE_PATH%\"" /f
reg add "HKCR\Directory\shell\Recolor\command" /v "" /t REG_SZ /d "\"%EXE_PATH%\" \"%%1\"" /f

:: Add to image files context menu
:: For PNG
reg add "HKCR\SystemFileAssociations\.png\shell\Recolor" /v "" /t REG_SZ /d "Recolor (recol)" /f
reg add "HKCR\SystemFileAssociations\.png\shell\Recolor" /v "Icon" /t REG_SZ /d "\"%EXE_PATH%\"" /f
reg add "HKCR\SystemFileAssociations\.png\shell\Recolor\command" /v "" /t REG_SZ /d "\"%EXE_PATH%\" \"%%1\"" /f

:: For JPG
reg add "HKCR\SystemFileAssociations\.jpg\shell\Recolor" /v "" /t REG_SZ /d "Recolor (recol)" /f
reg add "HKCR\SystemFileAssociations\.jpg\shell\Recolor" /v "Icon" /t REG_SZ /d "\"%EXE_PATH%\"" /f
reg add "HKCR\SystemFileAssociations\.jpg\shell\Recolor\command" /v "" /t REG_SZ /d "\"%EXE_PATH%\" \"%%1\"" /f

:: For JPEG
reg add "HKCR\SystemFileAssociations\.jpeg\shell\Recolor" /v "" /t REG_SZ /d "Recolor (recol)" /f
reg add "HKCR\SystemFileAssociations\.jpeg\shell\Recolor" /v "Icon" /t REG_SZ /d "\"%EXE_PATH%\"" /f
reg add "HKCR\SystemFileAssociations\.jpeg\shell\Recolor\command" /v "" /t REG_SZ /d "\"%EXE_PATH%\" \"%%1\"" /f

echo.
echo Installation complete!
echo You can now right-click on folders or PNG/JPG/JPEG images and select "Recolor (recol)".
pause
