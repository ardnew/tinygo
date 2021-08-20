package main

import (
	"errors"
	"flag"
	"fmt"
	"regexp"
	"strings"

	"github.com/tinygo-org/tinygo/tools/bitfields/util"
)

var (
	ErrArgument     = errors.New("invalid argument")
	ErrBitFieldSpec = fmt.Errorf("%w: bit field specification", ErrArgument)
	ErrRegisterSpec = fmt.Errorf("%w: register specification", ErrArgument)
	ErrPeriphSpec   = fmt.Errorf("%w: peripheral specification", ErrArgument)
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
	f.Var(&f.imports, "p", "Consider types and consts in each package `IMPORT`")
	f.Usage = f.usage
	f.Parse(args)
	// Remaining arguments, if any, are all peripheral/register/bit field specs.
	return f.Args()
}

func (f *flags) usage() {

	errf(Version() + `

Generate TinyGo source code containing types and methods for interacting with
individual bit fields within memory-mapped registers of integrated peripherals.

Source code is generated for all peripherals discovered if none are specified;
this is also true for registers and/or bit fields.

Peripherals, registers, and bit fields are located in packages at import path(s)
specified with flag -p (use flag -p multiple times to search multiple packages).
`)

	errf("Usage:")
	errf(indent(PROJECT), "[flags]", "[peripheral[:[register]:[bitfield,...]] ...]")
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
