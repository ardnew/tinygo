package main

import (
	"flag"
	"fmt"
	"regexp"
	"strings"

	"github.com/tinygo-org/tinygo/tools/gen-register-hal/util"
)

var Flags = flags{FlagSet: flag.CommandLine}

type flags struct {
	*flag.FlagSet
	version bool
	target  string
	imports importsFlag
}

type importsFlag struct{ path []string }

func (i *importsFlag) String() string {
	q := util.Smap(i.path, func(s string) string { return util.Qc(s, '"') })
	return "[" + strings.Join(q, ", ") + "]"
}

func (i *importsFlag) Set(path string) error {
	if i.path == nil {
		i.path = []string{}
	}
	for _, s := range i.path {
		if s == path {
			return nil // path already exists, just ignore this duplicate silently
		}
	}
	i.path = append(i.path, path)
	return nil
}

func (f *flags) Version() bool     { return f.version }
func (f *flags) Target() string    { return f.target }
func (f *flags) Imports() []string { return f.imports.path }

func (f *flags) parse(args []string) []string {
	f.BoolVar(&f.version, "version", false, "Print version and quit")
	f.StringVar(&f.target, "t", "", "Generate register descriptors for `TARGET`")
	f.Var(&f.imports, "p", "Use registers in each package `IMPORT`")
	f.Usage = f.usage
	f.Parse(args)
	// Remaining arguments, if any, are all peripheral type identifiers.
	return f.Args()
}

func (f *flags) usage() {

	errf(Version())
	errf("")
	errf("Generate TinyGo source code that adds memory-mapped register bitfield")
	errf("manipulation methods to the given peripheral(s) and target device.")
	errf("")
	errf("Methods are generated for all types found if no peripherals are specified.")
	errf("")
	errf("Peripherals are located in the import path(s) specified with flag -p, which")
	errf("may be given multiple times.")
	errf("")

	errf("Usage:")
	errf(indent(PROJECT), "[flags]", "[peripheral-type ...]")
	errf("")

	errf("Flags:")
	// Parse the first backtick-enclosed string in the given Flag's Usage string,
	// and append it to Flag's Name. Also returns the Usage string after removing
	// the matched surrounding backticks. This mimics package "flag"'s behavior.
	namedFlag := func(a *flag.Flag) (name string, usage string) {
		name, usage = a.Name, a.Usage
		if a.DefValue == "" {
			usage += " (REQUIRED)"
		} else {
			usage += fmt.Sprintf(" (default: %s)", a.DefValue)
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
			errf(indent(fmt.Sprintf("-%-*s%s", nameWidth, name, indent(usage))))
		})
}

func indent(s string) string { return "    " + s }
