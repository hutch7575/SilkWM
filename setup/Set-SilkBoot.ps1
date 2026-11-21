#requires -Version 5.1
#requires -RunAsAdministrator
[CmdletBinding()]
param([ValidateSet('Status','Hide','Restore')][string]$Mode='Status')
$ErrorActionPreference='Stop'
$edition=(Get-WindowsEdition -Online).Edition
if ($edition -notmatch '^(Enterprise|Education|IoTEnterprise)') {throw "Unbranded Boot is unavailable on $edition."}
$name='Client-EmbeddedBootExp'
$feature=Get-WindowsOptionalFeature -Online -FeatureName $name
$bcd=Join-Path $env:SystemRoot 'System32\bcdedit.exe'
$settings=& $bcd /enum '{globalsettings}'
if ($LASTEXITCODE -ne 0) {throw 'Cannot read boot settings.'}
$current=$settings | Where-Object {$_ -match '^\s*bootuxdisabled\s'}
# HKLM provides an administrator-owned record across restarts.
$key='HKLM:\SOFTWARE\Silk\Boot'
if ($Mode -eq 'Status') { [pscustomobject]@{Edition=$edition;Feature=$feature.State;BootUX=$current;Managed=(Test-Path $key)};return }
if ($Mode -eq 'Restore') {
    if (!(Test-Path $key)) {throw 'Silk has no saved change to restore.'}
    if ($current) {
        & $bcd /deletevalue '{globalsettings}' bootuxdisabled
        if ($LASTEXITCODE -ne 0) {throw 'Could not restore the boot setting.'}
    }
    Remove-Item -LiteralPath $key
    return
}
if ($feature.State -ne 'Enabled') {
    $result=Enable-WindowsOptionalFeature -Online -FeatureName $name -All -NoRestart
    if ($result.RestartNeeded) {Write-Output 'Restart Windows to finish enabling Unbranded Boot, then run Hide again.';return}
}
if ($current) {
    if (Test-Path $key) {Write-Output 'Silk boot setting already present.';return}
    throw 'An existing bootuxdisabled policy is already configured. Silk left it intact.'
}
$null=New-Item -Path $key -Force
& $bcd /set '{globalsettings}' bootuxdisabled on
if ($LASTEXITCODE -ne 0) {Remove-Item -LiteralPath $key;throw 'Windows rejected the boot setting.'}
