package parse

import (
	"bytes"
	"encoding/json"
	"errors"
	"go/ast"
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

type Parser struct {
	Config  Config
	Package map[string]*Package

	opts *compileopts.Config
	fset *token.FileSet
}

type Package struct {
	Json struct {
		Dir        string
		Name       string
		ImportPath string
		GoFiles    []string
		//CgoFiles []string
		//CFiles   []string
	}
	File []*ast.File
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
		Config:  c,
		Package: map[string]*Package{},
		fset:    token.NewFileSet(),
	}
	var err error

	parser.opts, err = builder.NewConfig(&compileopts.Options{Target: c.Target})
	if err != nil {
		return nil, err
	}

	return &parser, nil
}

func (p *Parser) Parse() error {

	var list = &bytes.Buffer{}

	cmd, err := loader.List(p.opts, []string{"-json"}, p.Config.Import)
	if err != nil {
		return err
	}
	cmd.Stdout = list
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	dec := json.NewDecoder(list)
	for {
		var k Package
		if err := dec.Decode(&k.Json); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		p.Package[k.Json.ImportPath] = &k
	}

	for _, pkg := range p.Package {
		pkg.File = []*ast.File{}
		for _, name := range pkg.Json.GoFiles {
			file, err := parser.ParseFile(p.fset, filepath.Join(pkg.Json.Dir, name),
				nil, parser.ParseComments)
			if err != nil {
				return err
			}
			pkg.File = append(pkg.File, file)
		}
	}

	return nil
}

// func mainPkg() (path string, err error) {
// 	// Create a temporary directory in TINYGOROOT whose name is the same as our
// 	// currently running executable (with a random suffix created by TempDir).
// 	// For example:
// 	//   /path/to/TINYGOROOT/gen-register-hal-843454839
// 	path, err = ioutil.TempDir(goenv.Get("TINYGOROOT"),
// 		filepath.Base(os.Args[0])+"-*")
// 	if err != nil {
// 		return "", err
// 	}
// 	// Write a minimal main package to appease the parser.
// 	err = ioutil.WriteFile(filepath.Join(path, "main.go"),
// 		[]byte(`package main;func main(){}`), 0644)
// 	if err != nil {
// 		return "", err
// 	}
// 	return
// }
