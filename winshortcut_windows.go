package main

import (
	"syscall"
	"unsafe"
)

const finderToggle = wmApp + 20
const catalogueReady = wmApp + 21
const winLeft = 0x5b
const winRight = 0x5c

var nextKey = user32.NewProc("CallNextHookEx")
var asyncKey = user32.NewProc("GetAsyncKeyState")
var sendKeys = user32.NewProc("SendInput")

type keyEvent struct {
	Key, Scan, Flags, Time uint32
	Extra                  uintptr
}
type keyInput struct {
	Kind        uint32
	_           uint32
	Key, Scan   uint16
	Flags, Time uint32
	_           uint32
	Extra       uintptr
	_           [8]byte
}
type winTap struct {
	key   uint32
	chord bool
	hook  uintptr
}

func (s *winTap) start() {
	if s.hook == 0 {
		s.hook = called(user32.NewProc("SetWindowsHookExW"), 13, shortcutCallback, called(pGetModuleHandle, 0), 0)
	}
}
func (s *winTap) stop() {
	if s.hook != 0 {
		called(user32.NewProc("UnhookWindowsHookEx"), s.hook)
	}
	*s = winTap{}
}

var shortcutCallback = syscall.NewCallback(onShortcut)

func onShortcut(code, msg, data uintptr) uintptr {
	pass := func() uintptr { return called(nextKey, 0, code, msg, data) }
	if int32(code) != 0 || app == nil {
		return pass()
	}
	e := (*keyEvent)(unsafe.Pointer(data))
	if e.Flags&0x10 != 0 {
		return pass()
	}
	s := &app.finder.tap
	if e.Key != winLeft && e.Key != winRight {
		if s.key != 0 {
			s.chord = true
		}
		return pass()
	}
	if msg == 0x100 || msg == 0x104 {
		if s.key == 0 {
			s.key = e.Key
			s.chord = false
			for _, k := range []uintptr{0x10, 0x11, 0x12, 1, 2, 4} {
				if called(asyncKey, k)&0x8000 != 0 {
					s.chord = true
					break
				}
			}
		} else if s.key != e.Key {
			s.chord = true
		}
		return pass()
	}
	tap := s.key == e.Key && !s.chord
	s.key = 0
	s.chord = false
	if !tap {
		return pass()
	}
	// Mark the modifier as used before releasing it. Windows receives a balanced
	// Win down/up pair and keeps ownership of every real Win+key chord.
	const neutral = 0xe8
	release := uint32(2) | (e.Flags & 1)
	keys := [3]keyInput{{Kind: 1, Key: neutral}, {Kind: 1, Key: neutral, Flags: 2}, {Kind: 1, Key: uint16(e.Key), Scan: uint16(e.Scan), Flags: release}}
	if called(sendKeys, 3, uintptr(unsafe.Pointer(&keys[0])), unsafe.Sizeof(keys[0])) != 3 {
		return pass()
	}
	called(pPostMessage, app.window, finderToggle, 0, 0)
	return 1
}
