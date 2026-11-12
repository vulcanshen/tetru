# tetru uninstaller for Windows
# Usage: irm https://raw.githubusercontent.com/vulcanshen/tetru/main/uninstall.ps1 | iex

$ErrorActionPreference = "Stop"

$installDir = "$env:LOCALAPPDATA\tetru"

if (-not (Test-Path "$installDir\tetru.exe")) {
    Write-Host "tetru not found in $installDir" -ForegroundColor Red
    exit 1
}

Remove-Item $installDir -Recurse -Force
Write-Host "removed $installDir" -ForegroundColor Green

# Remove from PATH
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -like "*$installDir*") {
    $newPath = ($userPath -split ";" | Where-Object { $_ -ne $installDir }) -join ";"
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    Write-Host "removed $installDir from PATH" -ForegroundColor Green
}

# Ask before removing one folder
function Ask-Remove($dir, $what) {
    if (-not (Test-Path $dir)) { return }
    $answer = Read-Host "Remove $what in $dir? [y/N]"
    if ($answer -eq "y" -or $answer -eq "Y") {
        Remove-Item $dir -Recurse -Force
        Write-Host "removed $dir" -ForegroundColor Green
    } else {
        Write-Host "kept $dir" -ForegroundColor Yellow
    }
}

# The settings: the keys to play with (tdp E3)
if ($env:TETRU__CONFIG) {
    $configDir = $env:TETRU__CONFIG
} elseif ($env:XDG_CONFIG_HOME) {
    $configDir = Join-Path $env:XDG_CONFIG_HOME "tetru"
} else {
    $configDir = Join-Path $env:USERPROFILE ".config\tetru"
}
Ask-Remove $configDir "tetru settings (the keys to play with)"

# The data: the scores and the game saved (tdp E3)
if ($env:TETRU__DATA) {
    $dataDir = $env:TETRU__DATA
} else {
    $dataDir = Join-Path $env:USERPROFILE ".tetru"
}
Ask-Remove $dataDir "tetru data (the scores and the game saved)"

Write-Host ""
Write-Host "tetru uninstalled. Restart your terminal for PATH changes to take effect." -ForegroundColor Green
