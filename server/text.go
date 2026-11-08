package main

import (
	"strings"
	"unicode"
)

func norm(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "ё", "е")
	return strings.TrimSpace(s)
}

func slug(s string) string {
	var out []rune
	lastDash := true
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out = append(out, r)
			lastDash = false
		} else if !lastDash {
			out = append(out, '-')
			lastDash = true
		}
	}
	return strings.Trim(string(out), "-")
}

func hasString(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func splitAlt(text string) []string {
	var result []string
	for _, part := range strings.Split(text, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func searchNorm(s string) string {
	s = norm(s)
	var out []rune
	lastSpace := true
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out = append(out, r)
			lastSpace = false
		} else if !lastSpace {
			out = append(out, ' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(string(out))
}
