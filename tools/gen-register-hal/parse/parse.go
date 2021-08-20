package parse

import (
	"bytes"
	"encoding/json"
	"errors"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"

	"github.com/tinygo-org/tinygo/builder"
	"github.com/tinygo-org/tinygo/compileopts"
	"github.com/tinygo-org/tinygo/loader"
)

var (
	ErrUnspecifiedTarget = errors.New("no TinyGo target (-t) specified")
	ErrNoImportPath      = errors.New("no package import path (-p) specified")
	ErrPackageNotFound   = errors.New("package not found")
)

// Parser is the primary runtime control structure that stores all of the
// requested peripherals and bitfields for each named package import path.
type Parser struct {
	Config  Config
	Package map[string]*Package
	opts    *compileopts.Config
}

// Config contains the user configuration options derived via command-line flags
// and/or process environment. Config fields are , and they
// select the peripheral types to include in the generated register interfaces.
type Config struct {

	// Target is used to select the appropriate TinyGo build tags for resolving Go
	// source files at a given import path. Target should be the full name of the
	// TinyGo target board (not a specific microcontroller or family), the same as
	// those used in the `tinygo -target` command-line flag.
	Target string

	// Import defines the Go package import paths to the target's SVD-generated
	// device interface descriptors, typically located in a "device" subdirectory
	// for the target's family of microcontrollers. For example:
	//  "device/sam", "device/stm32", "device/avr", or even "device/arm"
	//
	// These are relative paths using the same rules as standard Go's import path
	// resolution, but based on the selected target's TINYGOROOT.
	//
	// Note these should not be actual filesystem paths — except by coincidence —
	// and their paths should never end with a regular file name (with .go or any
	// other filename extension).
	//
	// It is not necessary for the resolved files at a given import path be among
	// the SVD-generated TinyGo source files. As long as the peripheral types,
	// bitmask constants, and so on..., all follow the same naming conventions and
	// file structure, then the AST should be handled appropriately. This tool
	// was, nonetheless, designed based on the output of TinyGo's gen-device-svd.
	// So... Your Mileage May Vary.
	Import []string

	// Types defines the package-local type identifiers for those peripherals
	// whose memory-mapped registers shall have methods generated to manipulate
	// their individual bitfields.
	//
	// For example, if Target is "teensy40", Import contains "device/nxp", and we
	// want to generate interfaces for the UART peripheral ("LPUART_Type"), then
	// types should contain that peripheral type identifier alone. It should NOT
	// contain a package qualification, or an instance of that type:
	//   "LPUART_Type"       // ok
	//   "nxp.LPUART_Type"   // bad: includes package-qualifier
	//   "LPUART0"           // bad: is an instance of type LPUART_Type
	//   "nxp.LPUART0"       // bad: package-qualified instance (come on now...)
	Types []string
}

func New(c Config) (*Parser, error) {

	if c.Target == "" {
		return nil, ErrUnspecifiedTarget
	}

	if len(c.Import) == 0 {
		return nil, ErrNoImportPath
	}

	var p = Parser{
		Config:  c,
		Package: map[string]*Package{},
	}
	var err error

	p.opts, err = builder.NewConfig(&compileopts.Options{Target: c.Target})
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (p *Parser) Parse() error {

	var b = &bytes.Buffer{}

	c, err := loader.List(p.opts, []string{"-json"}, p.Config.Import)
	if err != nil {
		return err
	}
	c.Stdout = b
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return err
	}

	d := json.NewDecoder(b)
	for {
		var k Package
		if err := d.Decode(&k.Json); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		k.Periph = map[string]*Periph{}
		for _, typ := range p.Config.Types {
			k.Periph[periphIdent(typ)] = &Periph{}
		}
		p.Package[k.Json.ImportPath] = &k
	}

	for _, k := range p.Package {
		fset := token.NewFileSet()
		for _, name := range k.Json.GoFiles {
			file, err := parser.ParseFile(fset, filepath.Join(k.Json.Dir, name),
				nil, parser.ParseComments)
			if err != nil {
				return err
			}
			k.scan(file)
		}
	}

	return nil
}
