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
    # Use the packaged desktop artwork for a continuous sign-in background.
    $Image = Join-Path $PSScriptRoot '..\assets\midnight-silk.jpg'

}
$path = (Get-Item -LiteralPath $Image).FullName
$file = Wait-StorageFile ([Windows.Storage.StorageFile]::GetFileFromPathAsync($path))
$action = [Windows.System.UserProfile.LockScreen]::SetImageFileAsync($file)
$method = $methods | Where-Object { $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncAction' } | Select-Object -First 1
$method.Invoke($null,@($action)).GetAwaiter().GetResult()
Write-Output 'Lock-screen image applied. In Settings > Personalisation > Lock screen, enable the option to show the picture on the sign-in screen.'
