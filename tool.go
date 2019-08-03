package main

import (
	"strings"
	"time"
)

func cleanStrSlice(ss []string) []string {
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		s = strings.TrimSpace(s)
		if len(s) != 0 {
			out = append(out, s)
		}
	}
	return out
}

func debounce(dur time.Duration, in chan interface{}, cb func()) {
	update := false
	timer := time.NewTimer(dur)
	for {
		select {
		case _ = <-in:
			update = true
			timer.Reset(dur)
		case <-timer.C:
			if update {
				cb()
			}
		}
	}
}

func shortenID(str string) string {
	if len(str) > 16 {
		return str[0:16]
	}
	return str
}
