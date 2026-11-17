param(
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[a-p]{32}$')]
    [string]$ExtensionId,
    [string]$UserSkins = "$env:LOCALAPPDATA\WarThunder\UserSkins"
)

$ErrorActionPreference = 'Stop'
$source = Join-Path $PSScriptRoot 'wt-live-host.exe'
if (-not (Test-Path -LiteralPath $source -PathType Leaf)) {
    throw 'wt-live-host.exe is missing. Download the complete repository or release package.'
}

$resolved = [IO.Path]::GetFullPath($UserSkins)
if ([IO.Path]::GetFileName($resolved.TrimEnd('\')) -ine 'UserSkins') {
    throw 'Specify the UserSkins folder, not the WarThunder or lang folder.'
}
if (-not (Test-Path -LiteralPath $resolved -PathType Container)) {
    New-Item -ItemType Directory -Path $resolved -Force | Out-Null
}

$target = Join-Path $env:LOCALAPPDATA 'WtLiveSkinInstaller'
New-Item -ItemType Directory -Path $target -Force | Out-Null
$hostExe = Join-Path $target 'wt-live-host.exe'
Copy-Item -LiteralPath $source -Destination $hostExe -Force

$configPath = Join-Path $target 'config.json'
$ids = @($ExtensionId)
if (Test-Path -LiteralPath $configPath) {
    $old = Get-Content -LiteralPath $configPath -Raw | ConvertFrom-Json
    $ids += @($old.extension_ids)
}
$ids = @($ids | Where-Object { $_ -match '^[a-p]{32}$' } | Select-Object -Unique)
$config = @{ extension_ids = $ids; user_skins = $resolved }
$manifest = @{
    name = 'com.wtlive.skin_installer'
    description = 'WT Live Skin Installer helper'
    path = $hostExe
    type = 'stdio'
    allowed_origins = @($ids | ForEach-Object { "chrome-extension://$_/" })
}
$utf8 = New-Object System.Text.UTF8Encoding($false)
[IO.File]::WriteAllText($configPath, ($config | ConvertTo-Json -Depth 4), $utf8)
$manifestPath = Join-Path $target 'com.wtlive.skin_installer.json'
[IO.File]::WriteAllText($manifestPath, ($manifest | ConvertTo-Json -Depth 4), $utf8)

foreach ($browser in @('Google\Chrome', 'Vivaldi')) {
    $key = "HKCU\Software\$browser\NativeMessagingHosts\com.wtlive.skin_installer"
    & reg.exe add $key /ve /t REG_SZ /d $manifestPath /f | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "Could not register native host for $browser" }
}

Write-Host "Helper installed for extension $ExtensionId"
Write-Host "Skins will be saved to $resolved"
Write-Host 'Restart the browser, then click Install via helper on WT Live.'
