# Silk

A configurable desktop for Windows 10 and 11, built on the original SilkWM tiler. Silk samples your Windows wallpaper and carries its accent into the ribbon mark, tiling indicator, menus and desktop reveal. The default layout pairs a slim top bar with a master window and a stack beside it. The shared theme and minimal desktop take inspiration from [osyx](https://github.com/rccyx/osyx).

## Run Silk

Extract the Windows x64 build to a permanent folder and launch **Silk.exe** as your normal user. Click the top bar or tray icon to open its menu. Pause or exit restores the captured positions of managed windows. To start at sign-in, run `.\setup\Set-SilkStartup.ps1` in Windows PowerShell from the extracted folder. Add `-Remove` to remove the startup shortcut. Exit the old SilkWM prototype before starting Silk.

Choose **Use Silk wallpaper** in the bar menu to apply the included Midnight Silk artwork, or keep your current wallpaper. Wallpaper changes are sampled in the background. The tint follows the Windows system wallpaper and uses your transition colour if sampling fails. Silk honours the Windows animation setting.

## Customise your desktop

Choose **Edit profile** from the menu, save, then choose **Reload profile**. Silk creates the file in `%APPDATA%\Silk\profile.json`. Omitted fields use defaults. An invalid reload leaves the current desktop running and explains the error.

| Setting | Default | Accepted values |
| --- | --- | --- |
| gap | 12 | 0–100 physical pixels between tiles |
| masterPercent | 60 | 10–90 percent of the available row |
| motionMs | 180 | 0–1500; zero snaps immediately |
| barHeight | 40 | 0–100 logical pixels; zero hides and releases the bar |
| widgets | ["silk","status","date","clock"] | Ordered list of silk, status, clock, date |
| accent | auto | Sample the wallpaper, or use a fixed #RRGGBB colour. Silk adjusts accent brightness for legibility. |
| surface / text | #10131B / #E8EBF2 | Bar background and label colour, #RRGGBB |
| transition / fadeMs | #195AB4 / 550 | Overlay colour and duration, 0–3000 ms; zero disables the fade |

The primary monitor's bar reserves desktop space through the Windows AppBar API. Tiling uses each monitor's work area, skips hidden, minimised and borderless fullscreen windows, and eases ordinary windows into position. Apps with minimum sizes or elevated permissions can resist tiling. **Preview desktop fade** plays the session-entry transition. **Restart Windows** asks for confirmation and plays the transition into the Windows restart request. Restarts started outside Silk follow the usual Windows path.

## Sign-in and boot

Run the scripts below in Windows PowerShell 5.1. The desktop executable needs no elevation.

| Stage | How to customise it |
| --- | --- |
| Your lock and sign-in picture | Run `.\setup\Set-SilkLockScreen.ps1` to apply the bundled Midnight Silk artwork, or pass `-Image C:\Pictures\lock.jpg`. Enable **Show the lock screen background picture on the sign-in screen** in Windows Settings. Select a different picture there to reset. |
| Signed-out machine default | On Enterprise, Education and IoT Enterprise, run `.\setup\Set-SilkSignIn.ps1 -Image C:\Pictures\lock.jpg` as administrator. This sets the Windows policy for the default lock and logon image. Use `-Restore` to remove Silk's policy. Domain policy can override local settings. |
| Session handover | Silk renders the blue fade when its startup shortcut runs and after an unlock notification. Windows controls the secure desktop and timing of the handover. Startup apps may be delayed, so the desktop can appear briefly first. Windows supplies password, PIN and Hello controls. |
| Before sign-in | On an eligible Enterprise, Education or IoT Enterprise installation, run `.\setup\Set-SilkBoot.ps1 Status` as administrator. `Hide` enables Unbranded Boot and suppresses standard Windows boot visuals. If the optional feature needs a restart, run Hide again afterwards. `Restore` removes Silk's boot setting; the optional component stays installed. Check that you have your BitLocker recovery key before editing boot settings. |
| Restart | Silk owns its initial restart fade. Windows owns the subsequent restart, servicing and recovery screens. Microsoft documents UEFI BGRT as the supported route for an OEM startup logo. Unbranded Boot offers suppression only. |

[Microsoft Unbranded Boot](https://learn.microsoft.com/windows/configuration/unbranded-boot/) · [Microsoft Custom Logon](https://learn.microsoft.com/windows/configuration/custom-logon/)

## Build

Use Go 1.23 or newer. The supported targets are Windows x64 and ARM64.

```powershell
go test ./...
go build -trimpath -ldflags="-H windowsgui -s -w" -o Silk.exe .
```

The Windows workflow builds the x64 executable, checks setup script syntax, launches Silk, checks for its bar window, requests a clean exit and packages the binary with the scripts.
