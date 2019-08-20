package main

import (
	"strconv"
	"strings"
	"time"
)

func cleanServiceName(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func cleanServicePort(s string) (ret int) {
	ret, _ = strconv.Atoi(strings.TrimSpace(s))
	return
}

func cleanServiceTags(ss []string) []string {
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		s = strings.ToLower(strings.TrimSpace(s))
		if len(s) != 0 {
			out = append(out, s)
		}
	}
	return out
}

func cleanServiceCheck(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func cleanContainerLabel(s string) string {
	return strings.TrimSpace(s)
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
	if len(str) > 24 {
		return str[0:12] + "-" + str[12:24]
	}
	return str
}
