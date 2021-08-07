package main

import (
	"fmt"
	"os"
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

	path, periph := Flags.parse(os.Args[1:])

	if Flags.Version() {
		fmt.Println(Version())
	} else {
		fmt.Fprintf(Flags.Out(), "path = %q\nperiph = %#v\n", path, periph)
	}
}
