#requires -Version 5.1
[CmdletBinding()]
param([string]$Image)
$ErrorActionPreference = 'Stop'
# WinRT personalisation runs under the interactive user's identity.
Add-Type -AssemblyName System.Runtime.WindowsRuntime
$null = [Windows.Storage.StorageFile,Windows.Storage,ContentType=WindowsRuntime]
$null = [Windows.System.UserProfile.LockScreen,Windows.System.UserProfile,ContentType=WindowsRuntime]
$methods = [System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object {
    $_.Name -eq 'AsTask' -and $_.GetParameters().Count -eq 1
}
function Wait-StorageFile($Operation) {
    $method = $methods | Where-Object {
        $_.IsGenericMethod -and $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncOperation`1'
    } | Select-Object -First 1
    $task = $method.MakeGenericMethod([Windows.Storage.StorageFile]).Invoke($null, @($Operation))
    $task.GetAwaiter().GetResult()
}
if (!$Image) {
    $dir = Join-Path $env:APPDATA 'Silk'
    $null = New-Item -ItemType Directory -Force $dir
    $profileFile = Join-Path $dir 'profile.json'
    $color = '#195AB4'
    if (Test-Path -LiteralPath $profileFile) {
        $custom = (Get-Content -Raw -Encoding UTF8 $profileFile | ConvertFrom-Json).transition
        if ($custom) { $color = $custom }
    }
    if ($color -notmatch '^#[a-fA-F0-9]{6}$') { throw 'Expected a #RRGGBB transition colour.' }
    Add-Type -AssemblyName System.Drawing
    $Image = Join-Path $dir 'silk-lock.png'
    $bitmap = New-Object Drawing.Bitmap 1920,1080
    $graphics = [Drawing.Graphics]::FromImage($bitmap)
    try {
        $graphics.Clear([Drawing.ColorTranslator]::FromHtml($color))
        $bitmap.Save($Image,[Drawing.Imaging.ImageFormat]::Png)
    } finally { $graphics.Dispose(); $bitmap.Dispose() }
}
$path = (Get-Item -LiteralPath $Image).FullName
$file = Wait-StorageFile ([Windows.Storage.StorageFile]::GetFileFromPathAsync($path))
$action = [Windows.System.UserProfile.LockScreen]::SetImageFileAsync($file)
$method = $methods | Where-Object { $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncAction' } | Select-Object -First 1
$method.Invoke($null,@($action)).GetAwaiter().GetResult()
Write-Output 'Lock-screen image applied. In Settings > Personalisation > Lock screen, enable the option to show the picture on the sign-in screen.'
