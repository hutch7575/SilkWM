package main

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

type appEntry struct {
	Name, Target string
	Shortcut     bool
}

func matches(apps []appEntry, query string) []appEntry {
	type hit struct {
		app   appEntry
		score int
	}
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	words := strings.Fields(q)
	var hits []hit
	for _, a := range apps {
		n := strings.ToLower(a.Name)
		score := 0
		for _, w := range words {
			at := strings.Index(n, w)
			if at < 0 {
				score = -1
				break
			}
			score += min(at, 40) + 50
			prev, _ := utf8.DecodeLastRuneInString(n[:at])
			if at == 0 || unicode.IsSpace(prev) {
				score -= 48
			}
		}
		if score < 0 {
			continue
		}
		if n == q {
			score = -2
		} else if strings.HasPrefix(n, q) {
			score -= 1
		}
		hits = append(hits, hit{a, score})
	}
	sort.SliceStable(hits, func(i, j int) bool {
		a, b := hits[i], hits[j]
		if a.score != b.score {
			return a.score < b.score
		}
		return strings.ToLower(a.app.Name) < strings.ToLower(b.app.Name)
	})
	out := make([]appEntry, len(hits))
	for i, h := range hits {
		out[i] = h.app
	}
	return out
}
