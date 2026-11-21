package main

import (
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

var systemParameters = user32.NewProc("SystemParametersInfoW")

type wallSample struct {
	path  string
	stamp time.Time
	color uintptr
	valid bool
}
type wallpaperWatcher struct {
	previous  uintptr
	changed   time.Time
	last      wallSample
	pending   chan wallSample
	lastCheck time.Time
}

func activeWallpaper() string {
	var buf [32768]uint16
	if called(systemParameters, 0x73, uintptr(len(buf)), uintptr(unsafe.Pointer(&buf[0])), 0) != 0 {
		p := syscall.UTF16ToString(buf[:])
		if p != "" {
			return p
		}
	}
	return ""
}
func (w *wallpaperWatcher) refresh(force bool) {
	if w.pending != nil {
		select {
		case r := <-w.pending:
			w.pending = nil
			fallback, _ := colorRef(app.profile.Transition)
			before := w.colorAt(fallback)
			old := w.last
			w.last = r
			if app.profile.Accent == "auto" && (old.valid != r.valid || old.color != r.color) {
				w.previous = before
				w.changed = time.Now()
				if !app.surfaces.reduceMotion && app.profile.MotionMS > 0 {
					called(pSetTimer, app.window, 5, 16, 0)
				}
			}
			if app.surfaces.bar != 0 {
				called(invalidate, app.surfaces.bar, 0, 0)
			}
		default:
			return
		}
	}
	if !force && time.Since(w.lastCheck) < 3*time.Second {
		return
	}
	w.lastCheck = time.Now()
	w.pending = make(chan wallSample, 1)
	go func(result chan<- wallSample, last wallSample) {
		p := activeWallpaper()
		r := wallSample{path: p}
		if p != "" {
			info, e := os.Stat(p)
			if e == nil {
				r.stamp = info.ModTime()
				if last.valid && r.path == last.path && r.stamp == last.stamp {
					result <- last
					return
				}
			}
			col, e := sampleFile(p)
			if e != nil {
				col, e = sampleFile(filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Themes", "TranscodedWallpaper"))
			}
			r.color, r.valid = col, e == nil
		}
		result <- r
	}(w.pending, w.last)
}
func setSilkWallpaper() error {
	exe, e := os.Executable()
	if e != nil {
		return e
	}
	path := filepath.Join(filepath.Dir(exe), "assets", "midnight-silk.jpg")
	if _, e = os.Stat(path); e != nil {
		return e
	}
	ok, _, e := systemParameters.Call(0x14, 0, uintptr(unsafe.Pointer(ptr(path))), 3)
	if ok == 0 {
		return e
	}
	return nil
}

func (w *wallpaperWatcher) colorAt(fallback uintptr) uintptr {
	target := fallback
	if w.last.valid {
		target = w.last.color
	}
	if w.changed.IsZero() {
		return target
	}
	t := progress(time.Since(w.changed), app.surfaces.duration(min(220, app.profile.MotionMS)))
	return blend(w.previous, target, t)
}
func (w *wallpaperWatcher) frame() {
	if time.Since(w.changed) >= time.Duration(app.surfaces.duration(min(220, app.profile.MotionMS)))*time.Millisecond {
		called(pKillTimer, app.window, 5)
		w.changed = time.Time{}
	}
	if app.surfaces.bar != 0 {
		called(invalidate, app.surfaces.bar, 0, 0)
	}
}
