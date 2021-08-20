package util

import (
	"os"
	"strings"

	"github.com/davecgh/go-spew/spew"
)

func Smap(s []string, f func(string) string) []string {
	t := make([]string, len(s))
	for i, a := range s {
		t[i] = f(a)
	}
	return t
}

func IsInSlice(slice []string, s string) bool {
	for _, c := range slice {
		if c == s {
			return true
		}
	}
	return false
}

func Qc(s string, q rune) string { return Qs(s, string(q)) }
func Qs(s, q string) string {
	for len(s) >= 2*len(q) && strings.HasPrefix(s, q) && strings.HasSuffix(s, q) {
		s = strings.TrimSuffix(strings.TrimPrefix(s, q), q)
	}
	return q + s + q
}

func Dump(a ...interface{}) {
	c := spew.NewDefaultConfig()
	c.Indent = "    "
	c.MaxDepth = 0
	c.Fdump(os.Stderr, a...)
}
