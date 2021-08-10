package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/tinygo-org/tinygo/tools/gen-register-hal/parse"
	"github.com/tinygo-org/tinygo/tools/gen-register-hal/util"
)

var (
	PROJECT   string
	IMPORT    string
	VERSION   string
	BUILDTIME string
	PLATFORM  string
)

func Version() string {
	return fmt.Sprintf("%s %s (%s) %s", PROJECT, VERSION, PLATFORM, BUILDTIME)
}

func main() {

	typ := Flags.parse(os.Args[1:])

	if Flags.Version() {
		fmt.Println(Version())
	} else {
		config := parse.Config{
			Target: Flags.Target(),
			Import: Flags.Imports(),
			Types:  typ,
		}
		par, err := parse.New(config)
		if err != nil {
			halt(err)
		}
		util.Dump(par.Package)
	}
}

func halt(err error) {

	errf("error: %s", err.Error())

	code := 127
	switch {
	case errors.Is(err, parse.ErrUnspecifiedTarget):
		code = 2
	case errors.Is(err, parse.ErrNoImportPath):
		code = 3
	case errors.Is(err, parse.ErrPackageNotFound):
		code = 4
	}
	os.Exit(code)
}

func logf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, format, args...)
	fmt.Fprintln(os.Stdout)
}

func errf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	fmt.Fprintln(os.Stderr)
}
