#requires -Version 5.1
#requires -RunAsAdministrator
[CmdletBinding(DefaultParameterSetName='Apply')]
param(
    [Parameter(Mandatory,ParameterSetName='Apply')][string]$Image,
    [Parameter(Mandatory,ParameterSetName='Restore')][switch]$Restore
)
$ErrorActionPreference='Stop'
$key='HKLM:\SOFTWARE\Policies\Microsoft\Windows\Personalization'
$name='LockScreenImage'
$folder=Join-Path $env:ProgramFiles 'Silk\Branding'
$target=Join-Path $folder 'SignIn.png'
$current=Get-ItemPropertyValue -LiteralPath $key -Name $name -ErrorAction SilentlyContinue
if ($Restore) {
    if ($current -eq $target) {Remove-ItemProperty -LiteralPath $key -Name $name}
    if (Test-Path -LiteralPath $target) {Remove-Item -LiteralPath $target}
    return
}
$edition=(Get-WindowsEdition -Online).Edition
if ($edition -notmatch '^(Enterprise|Education|IoTEnterprise)') {throw "Machine-wide sign-in artwork is unavailable on $edition. Use Set-SilkLockScreen.ps1 for your account."}
if ($null -ne $current -and $current -ne $target) {throw 'Another lock-screen picture policy is already configured.'}
Add-Type -AssemblyName System.Drawing
$null=New-Item -ItemType Directory -Path $folder -Force
$temp=Join-Path $folder ([Guid]::NewGuid().ToString()+'.png')
try {
    $source=[Drawing.Image]::FromFile((Get-Item -LiteralPath $Image).FullName)
    try {$source.Save($temp,[Drawing.Imaging.ImageFormat]::Png)} finally {$source.Dispose()}
    if (Test-Path -LiteralPath $target) {[IO.File]::Replace($temp,$target,$null)} else {[IO.File]::Move($temp,$target)}
    if (!(Test-Path $key)) {$null=New-Item $key -Force}
    $null=New-ItemProperty -LiteralPath $key -Name $name -Value $target -PropertyType String -Force
} finally {if (Test-Path -LiteralPath $temp) {Remove-Item -LiteralPath $temp}}
