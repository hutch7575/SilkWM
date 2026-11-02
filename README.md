<div align="center">

<img src="./resources/assets/logo.svg" width="260" alt="SilkWM logo" />

</div>

---

<div align="center">

**An actually smooth window manager.**

</div>

## Prototype (Windows)

An early, experimental auto-tiling prototype. **This version does not have animations yet.**

- Automatically arranges ordinary windows in a **60% master / stacked-right** layout.
- Tiles independently on each monitor, respecting the taskbar's work area.
- Ignores minimized, hidden, tool and borderless fullscreen windows.
- Lives in the notification area (system tray). Right-click the SilkWM icon to **Pause tiling**, **Rearrange windows**, or **Exit**.
- On pause or exit, attempts to restore windows to their original locations.
- Does not require administrator access, an installer, or external runtime dependencies.

**Run:** download `SilkWM.exe` from a build or compile it yourself, launch it, and find its icon in the system tray (possibly under the `^` overflow menu). To exit, use the tray menu. SilkWM does not run on Windows startup automatically.

**Build (Windows, Go 1.23+):**

```powershell
go build -trimpath -ldflags="-H windowsgui -s -w" -o SilkWM.exe .
```

**Known limitations:** prototype only; no workspaces, resize shortcuts, per-app rules or animations. Some elevated or protected applications can't be repositioned by a normal-user process. Windows that enforce a minimum size may not fit their assigned tile. Consider testing with a few ordinary apps first. No system settings are modified.
