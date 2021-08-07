package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/tinygo-org/tinygo/goenv"
)

func Indent(s string) string { return "    " + s }

type flags struct {
	*flag.FlagSet
	out     io.Writer
	version bool
	goroot  string
}

var Flags = flags{FlagSet: flag.CommandLine, out: os.Stderr}

func (f *flags) Out() io.Writer { return f.out }
func (f *flags) Version() bool  { return f.version }
func (f *flags) Goroot() string { return f.goroot }

func (f *flags) parse(args []string) (path string, periph []string) {
	f.BoolVar(&f.version, "version", false, "Print version and quit")
	f.StringVar(&f.goroot, "r", goenv.Get("TINYGOROOT"), "Use `PATH` as TINYGOROOT")
	f.Usage = f.usage
	f.Parse(args)

	path = f.Arg(0)
	if f.NArg() > 1 {
		periph = f.Args()[1:]
	} else {
		periph = []string{}
	}
	return
}

func (f *flags) usage() {

	fmt.Fprintln(f.out, Version())
	fmt.Fprintln(f.out)

	fmt.Fprintln(f.out, "Usage:")
	fmt.Fprintln(f.out, Indent(PROJECT), "[flags]", "pkg-import-path", "[peripheral-type ...]")
	fmt.Fprintln(f.out)

	fmt.Fprintln(f.out, "Flags:")
	// Parse the first backtick-enclosed string in the given Flag's Usage string,
	// and append it to Flag's Name. Also returns the Usage string after removing
	// the matched surrounding backticks. This mimics package "flag"'s behavior.
	namedFlag := func(a *flag.Flag) (name string, usage string) {
		name, usage = a.Name, a.Usage
		if a.DefValue != "" {
			usage += fmt.Sprintf(" (default: %v)", a.DefValue)
		}
		r := regexp.MustCompile("^([^`]*)`([^`]+)`")
		s := r.FindStringSubmatch(a.Usage)
		if s != nil {
			name += " " + s[2]
			usage = r.ReplaceAllString(usage, "$1$2")
		}
		return
	}
	// Find the width of the widest flag name.
	nameWidth := 1
	f.VisitAll(
		func(a *flag.Flag) {
			name, _ := namedFlag(a)
			if nameWidth < len(name) {
				nameWidth = len(name)
			}
		})
	// Print each flag's usage description
	f.VisitAll(
		func(a *flag.Flag) {
			name, usage := namedFlag(a)
			fmt.Fprintln(f.out, Indent(fmt.Sprintf("-%-*s%s", nameWidth, name, Indent(usage))))
		})
}
