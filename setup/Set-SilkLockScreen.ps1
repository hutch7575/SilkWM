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
    $dir = Join-Path $env:LOCALAPPDATA 'Silk'
    $null = New-Item -ItemType Directory -Force -Path $dir
    $Image = Join-Path $dir 'Silk-SignIn.jpg'
    $exe = Join-Path $PSScriptRoot '..\Silk.exe'
    $render = Start-Process -FilePath $exe -ArgumentList ('--signin-art "' + $Image + '"') -Wait -PassThru
    if ($render.ExitCode -ne 0) {throw 'Silk could not render the sign-in artwork.'}
}
$path = (Get-Item -LiteralPath $Image).FullName
$file = Wait-StorageFile ([Windows.Storage.StorageFile]::GetFileFromPathAsync($path))
$action = [Windows.System.UserProfile.LockScreen]::SetImageFileAsync($file)
$method = $methods | Where-Object { $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncAction' } | Select-Object -First 1
$method.Invoke($null,@($action)).GetAwaiter().GetResult()
Write-Output 'Lock-screen image applied. In Settings > Personalisation > Lock screen, enable the option to show the picture on the sign-in screen.'
