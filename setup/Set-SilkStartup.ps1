[CmdletBinding()]
param([switch]$Remove,[string]$Exe=(Join-Path $PSScriptRoot '..\Silk.exe'))
$ErrorActionPreference='Stop'
$path=Join-Path ([Environment]::GetFolderPath('Startup')) 'Silk.lnk'
if ($Remove) {if (Test-Path -LiteralPath $path) {Remove-Item -LiteralPath $path};return}
$file=Get-Item -LiteralPath $Exe
$link=(New-Object -ComObject WScript.Shell).CreateShortcut($path)
$link.TargetPath=$file.FullName
$link.WorkingDirectory=$file.DirectoryName
$link.Save()
