package parse

import (
	"errors"
	"fmt"
	"go/types"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/tinygo-org/tinygo/builder"
	"github.com/tinygo-org/tinygo/compileopts"
	"github.com/tinygo-org/tinygo/compiler"
	"github.com/tinygo-org/tinygo/goenv"
	"github.com/tinygo-org/tinygo/loader"

	"github.com/tinygo-org/tinygo/tools/gen-register-hal/util"
)

var (
	ErrUnspecifiedTarget = errors.New("no TinyGo target (-t) specified")
	ErrNoImportPath      = errors.New("no package import path (-p) specified")
	ErrPackageNotFound   = errors.New("package not found")
)

type Parser struct {
	Program *loader.Program
	Package map[string]*loader.Package
	mainPkg string
}

type Config struct {
	Target string
	Import []string
	Types  []string
}

func New(c Config) (*Parser, error) {

	if c.Target == "" {
		return nil, ErrUnspecifiedTarget
	}

	if len(c.Import) == 0 {
		return nil, ErrNoImportPath
	}

	var parser = Parser{
		Package: map[string]*loader.Package{},
	}

	conf, err := builder.NewConfig(&compileopts.Options{Target: c.Target})
	if err != nil {
		return nil, err
	}

	mainDir := goenv.Get("TINYGOROOT")
	dir, err := ioutil.TempDir(mainDir, filepath.Base(os.Args[0])+"-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	parser.mainPkg = filepath.Join(dir, "main.go")
	if err := ioutil.WriteFile(parser.mainPkg, gomain(), 0644); err != nil {
		return nil, err
	}

	mach, err := compiler.NewTargetMachine(
		&compiler.Config{
			Triple:          conf.Triple(),
			CPU:             conf.CPU(),
			Features:        conf.Features(),
			GOOS:            conf.GOOS(),
			GOARCH:          conf.GOARCH(),
			CodeModel:       conf.CodeModel(),
			RelocationModel: conf.RelocationModel(),

			Scheduler:          conf.Scheduler(),
			FuncImplementation: conf.FuncImplementation(),
			AutomaticStackSize: conf.AutomaticStackSize(),
			DefaultStackSize:   conf.Target.DefaultStackSize,
			NeedsStackObjects:  conf.NeedsStackObjects(),
			Debug:              true,
			LLVMFeatures:       conf.LLVMFeatures(),
		})
	if err != nil {
		return nil, err
	}

	parser.Program, err = loader.Load(conf,
		append([]string{filepath.Dir(parser.mainPkg)}, c.Import...),
		conf.ClangHeaders, types.Config{Sizes: compiler.Sizes(mach)})
	if err != nil {
		return nil, err
	}

	for path, pkg := range parser.Program.Packages {
		if util.IsInSlice(c.Import, path) {
			parser.Package[path] = pkg
		}
	}

	var missing string
	for _, path := range c.Import {
		if _, ok := parser.Package[path]; !ok {
			if len(missing) > 0 {
				missing += ", "
			}
			missing += util.Qc(path, '"')
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: %q", ErrPackageNotFound, missing)
	}

	return &parser, parser.Program.Parse()
}

func gomain() []byte {
	return []byte(`
package main

func main() {}
	`)
}
