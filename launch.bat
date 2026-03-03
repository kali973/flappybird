@echo off
setlocal EnableDelayedExpansion
title Flappy Bird Pro - Launcher

echo.
echo  +----------------------------------------------------------+
echo  ^|        FLAPPY BIRD PRO  --  Go Edition                  ^|
echo  ^|        Script de lancement Windows                      ^|
echo  +----------------------------------------------------------+
echo.

:: Se placer dans le dossier du script en premier
cd /d "%~dp0"

:: -----------------------------------------------------------
:: ETAPE 1 - Verifier si Go est installe
:: -----------------------------------------------------------
echo  [ .... ]  Verification de Go...

where go >nul 2>&1
if %errorlevel% == 0 (
    for /f "tokens=3" %%v in ('go version') do set GO_VER=%%v
    echo  [  OK  ]  Go !GO_VER! detecte
    goto :check_project
)

echo  [ WARN ]  Go non trouve -- installation en cours...
echo.

:: Methode 1 : winget
where winget >nul 2>&1
if %errorlevel% == 0 (
    echo  [ .... ]  Tentative via winget...
    winget install GoLang.Go --silent --accept-package-agreements --accept-source-agreements
    set "PATH=%PATH%;C:\Program Files\Go\bin"
    where go >nul 2>&1
    if %errorlevel% == 0 (
        echo  [  OK  ]  Go installe via winget
        goto :check_project
    )
)

:: Methode 2 : MSI direct
echo  [ .... ]  Telechargement du MSI Go 1.22...
set GO_MSI=%TEMP%\go_installer.msi
powershell -NoProfile -Command "[Net.ServicePointManager]::SecurityProtocol='Tls12'; Invoke-WebRequest -Uri 'https://go.dev/dl/go1.22.4.windows-amd64.msi' -OutFile '%GO_MSI%'"

if not exist "%GO_MSI%" (
    echo  [ FAIL ]  Telechargement echoue.
    echo   Installe Go manuellement : https://go.dev/dl/
    pause
    exit /b 1
)

echo  [ .... ]  Installation de Go -- une fenetre va apparaitre...
msiexec /i "%GO_MSI%" /passive /norestart
del "%GO_MSI%" >nul 2>&1
set "PATH=%PATH%;C:\Program Files\Go\bin"

where go >nul 2>&1
if %errorlevel% neq 0 (
    echo  [ FAIL ]  Installation Go echouee.
    echo   Installe Go manuellement : https://go.dev/dl/
    pause
    exit /b 1
)
echo  [  OK  ]  Go installe avec succes

:: -----------------------------------------------------------
:: ETAPE 2 - Verifier que le projet est present
:: -----------------------------------------------------------
:check_project
echo.
echo  [ .... ]  Verification du projet...

if not exist "go.mod" (
    echo  [ FAIL ]  go.mod introuvable.
    echo   Assure-toi que launch.bat est dans le dossier flappybird/
    pause
    exit /b 1
)
if not exist "main.go" (
    echo  [ FAIL ]  main.go introuvable.
    pause
    exit /b 1
)
echo  [  OK  ]  Fichiers du projet trouves

:: -----------------------------------------------------------
:: ETAPE 3 - Telecharger les dependances
:: -----------------------------------------------------------
echo.
echo  [ .... ]  Telechargement des dependances (Ebitengine)...
echo            Patience lors du premier lancement...

go get . >nul 2>&1
if %errorlevel% neq 0 (
    echo  [ WARN ]  go get echoue, tentative go mod tidy...
    go mod tidy
    if %errorlevel% neq 0 (
        echo  [ FAIL ]  Impossible de telecharger les dependances.
        echo   Verifie ta connexion Internet.
        pause
        exit /b 1
    )
)
echo  [  OK  ]  Dependances OK

:: -----------------------------------------------------------
:: ETAPE 4 - Compiler
:: -----------------------------------------------------------
echo.
echo  [ .... ]  Compilation...

set NEED_BUILD=1

if exist "flappybird.exe" (
    for %%F in ("main.go") do set SRC_DATE=%%~tF
    for %%F in ("flappybird.exe") do set BIN_DATE=%%~tF
    if "!SRC_DATE!" LEQ "!BIN_DATE!" (
        echo  [  OK  ]  Binaire a jour -- compilation ignoree
        set NEED_BUILD=0
    )
)

if !NEED_BUILD! == 1 (
    go build -ldflags="-H windowsgui" -o flappybird.exe .
    if %errorlevel% neq 0 (
        echo  [ WARN ]  Retry sans mode fenetre...
        go build -o flappybird.exe .
        if %errorlevel% neq 0 (
            echo  [ FAIL ]  Compilation echouee. Lancement via go run...
            goto :run_direct
        )
    )
    echo  [  OK  ]  Compilation reussie
)

:: -----------------------------------------------------------
:: ETAPE 5 - Lancer le jeu
:: -----------------------------------------------------------
:launch
echo.
echo  +------------------------------------------+
echo  ^|   FLAPPY BIRD PRO  --  Bon jeu !         ^|
echo  ^|                                          ^|
echo  ^|   ESPACE ou CLIC   = Sauter/Demarrer    ^|
echo  ^|   W ou fleche haut = Sauter             ^|
echo  ^|   Alt+F4           = Quitter            ^|
echo  +------------------------------------------+
echo.

flappybird.exe
exit /b 0

:run_direct
echo  [ .... ]  Lancement via go run (mode debug)...
go run .
exit /b 0
