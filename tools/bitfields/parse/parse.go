package parse

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tinygo-org/tinygo/builder"
	"github.com/tinygo-org/tinygo/compileopts"
	"github.com/tinygo-org/tinygo/loader"
)

var (
	ErrArgument          = errors.New("invalid argument")
	ErrFieldSpec         = fmt.Errorf("%w: bit field specification", ErrArgument)
	ErrRegisterSpec      = fmt.Errorf("%w: register specification", ErrArgument)
	ErrPeriphSpec        = fmt.Errorf("%w: peripheral specification", ErrArgument)
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

	// TODO: godoc
	Type typeSpec
}

type typeSpec map[string]map[string][]string

func New(c Config, arg ...string) (*Parser, error) {

	if c.Target == "" {
		return nil, ErrUnspecifiedTarget
	}

	if len(c.Import) == 0 {
		return nil, ErrNoImportPath
	}

	// Parse the remaining command-line arguments to populate our Config.
	if err := c.parse(arg...); err != nil {
		return nil, err
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
		for typ := range p.Config.Type {
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
			k.Scan(file)
		}
	}

	return nil
}

func (c *Config) parse(arg ...string) error {

	// TODO: replace manual string parsing with following regular expression:
	//   (?P<peripheral>[^\s:]+)(?::(?P<register>[^\s:]*)(?::(?P<bitfields>[^\s:]*))?)?
	c.Type = typeSpec{}
	for _, s := range arg {
		es := strings.Split(s, ":")
		bs := []string{}
		if len(es) > 2 {
			for _, b := range es[2:] {
				bs = append(bs, strings.Split(b, ",")...)
			}
		}
		rs := ""
		if len(es) > 1 {
			rs = es[1]
		}
		// Split will always return at least 1 element if sep is not empty.
		if cr, ok := c.Type[es[0]]; ok {
			// We already have this peripheral in the spec.
			// Check if we have this register included with that peripheral.
			if cb, ok := cr[rs]; ok {
				// We already have this peripheral register in the spec.
				// Append all bit fields that are not already included in the register.
				for _, b := range bs {
					exists := false
					for _, c := range cb {
						if exists = b == c; exists {
							break
						}
					}
					// Silently ignore any duplicate bit fields.
					if !exists {
						cb = append(cb, b)
					}
				}
				// Make sure the register has the updated bit field slice.
				cr[rs] = cb
			} else {
				// This is a new register added to the peripheral.
				cr[rs] = bs
			}
			// Make sure the spec has the updated register.
			c.Type[es[0]] = cr
		} else {
			// This is a new peripheral added to the spec.
			c.Type[es[0]] = map[string][]string{rs: bs}
		}
	}

	return nil
}
