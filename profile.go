package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Profile struct {
	Accent        string   `json:"accent"`
	Gap           int32    `json:"gap"`
	MasterPercent int32    `json:"masterPercent"`
	MotionMS      int      `json:"motionMs"`
	BarHeight     int32    `json:"barHeight"`
	Widgets       []string `json:"widgets"`
	Surface       string   `json:"surface"`
	Text          string   `json:"text"`
	Transition    string   `json:"transition"`
	FadeMS        int      `json:"fadeMs"`
}

func defaults() Profile {
	return Profile{Accent: "auto", Gap: 12, MasterPercent: 60, MotionMS: 180, BarHeight: 40, Widgets: []string{"silk", "status", "date", "clock"}, Surface: "#10131B", Text: "#E8EBF2", Transition: "#195AB4", FadeMS: 550}
}
func colorRef(s string) (uintptr, error) {
	if len(s) != 7 || s[0] != '#' {
		return 0, errors.New("expected #RRGGBB")
	}
	v, e := strconv.ParseUint(s[1:], 16, 24)
	return uintptr(v>>16 | v&0xff00 | (v&255)<<16), e
}
func parseProfile(b []byte) (Profile, error) {
	b = bytes.TrimPrefix(b, []byte{0xef, 0xbb, 0xbf})
	p := defaults()
	d := json.NewDecoder(strings.NewReader(string(b)))
	d.DisallowUnknownFields()
	if e := d.Decode(&p); e != nil {
		return p, e
	}
	if !json.Valid(b) || strings.TrimSpace(string(b)) == "null" {
		return p, errors.New("expected one JSON object")
	}
	if p.Gap < 0 || p.Gap > 100 || p.MasterPercent < 10 || p.MasterPercent > 90 || p.MotionMS < 0 || p.MotionMS > 1500 || p.BarHeight < 0 || p.BarHeight > 100 || p.FadeMS < 0 || p.FadeMS > 3000 {
		return p, errors.New("value outside supported range (see README)")
	}
	if p.Accent != "auto" {
		if _, e := colorRef(p.Accent); e != nil {
			return p, errors.New("accent must be auto or #RRGGBB")
		}
	}
	for _, v := range []string{p.Surface, p.Text, p.Transition} {
		if _, e := colorRef(v); e != nil {
			return p, fmt.Errorf("invalid colour %q", v)
		}
	}
	for _, v := range p.Widgets {
		switch v {
		case "silk", "status", "clock", "date":
		default:
			return p, fmt.Errorf("unknown widget %q", v)
		}
	}
	return p, nil
}
func profileFile() (string, error) {
	dir, e := os.UserConfigDir()
	if e != nil {
		return "", e
	}
	return filepath.Join(dir, "Silk", "profile.json"), nil
}
func readProfile() (Profile, error) {
	path, e := profileFile()
	if e != nil {
		return defaults(), e
	}
	b, e := os.ReadFile(path)
	if os.IsNotExist(e) {
		if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
			return defaults(), e
		}
		b, _ = json.MarshalIndent(defaults(), "", "  ")
		e = os.WriteFile(path, append(b, '\n'), 0600)
	}
	if e != nil {
		return defaults(), e
	}
	return parseProfile(b)
}
