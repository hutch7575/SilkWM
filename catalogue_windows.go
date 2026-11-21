package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func shortcuts() []appEntry {
	var out []appEntry
	seen := map[string]bool{}
	for _, root := range []string{os.Getenv("APPDATA"), os.Getenv("ProgramData")} {
		if root == "" {
			continue
		}
		root = filepath.Join(root, "Microsoft", "Windows", "Start Menu", "Programs")
		filepath.WalkDir(root, func(p string, d fs.DirEntry, e error) error {
			if e != nil || d.IsDir() || !strings.EqualFold(filepath.Ext(p), ".lnk") {
				return nil
			}
			rel, _ := filepath.Rel(root, p)
			key := strings.ToLower(rel)
			if !seen[key] {
				seen[key] = true
				out = append(out, appEntry{strings.TrimSuffix(d.Name(), filepath.Ext(p)), p, true})
			}
			return nil
		})
	}
	return out
}
func startApps(links []appEntry) []appEntry {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	exe := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	cmd := exec.CommandContext(ctx, exe, "-NoLogo", "-NoProfile", "-NonInteractive", "-Command",
		"[Console]::OutputEncoding=[Text.UTF8Encoding]::new($false); ConvertTo-Json -Compress -InputObject @(Get-StartApps -ErrorAction Stop | Select-Object Name,AppID)")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	b, e := cmd.Output()
	if e != nil {
		return links
	}
	var rows []struct{ Name, AppID string }
	if json.Unmarshal(bytes.TrimPrefix(b, []byte{0xef, 0xbb, 0xbf}), &rows) != nil {
		return links
	}
	names := map[string]bool{}
	var apps []appEntry
	for _, r := range rows {
		if r.Name != "" && r.AppID != "" && !strings.ContainsRune(r.AppID, 0) {
			apps = append(apps, appEntry{r.Name, r.AppID, false})
			names[strings.ToLower(r.Name)] = true
		}
	}
	for _, a := range links {
		if !names[strings.ToLower(a.Name)] {
			apps = append(apps, a)
		}
	}
	return apps
}
