package parse

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/davecgh/go-spew/spew"
)

type Package struct {
	Json struct {
		Dir        string
		Name       string
		ImportPath string
		GoFiles    []string
	}
	Periph map[string]*Periph
}

// scan performs a breadth-first traversal of the Go AST of a single file in
// the receiver Package, collecting the registers from each requested peripheral
// type, and the bit mask constants associated with those registers.
//
// The peripheral type must be declared at the file's outer-most lexical scope.
func (p *Package) scan(f *ast.File) {

	// First locate our type definitions and gather all of their registers.
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			switch d.Tok {
			case token.CONST:
			case token.VAR:
			case token.TYPE:
				// Found a "type (...)" block. Inspect each of its contained type
				// specifications to determine if any name matches one of the types
				// requested by the user. If so, traverse that type specification for
				// each of the registers contained within.
				for _, spec := range d.Specs {
					switch t := spec.(type) {
					case *ast.TypeSpec:
						if !t.Name.IsExported() {
							continue
						}
						i := periphIdent(t.Name.Name)
						if per, ok := p.Periph[i]; ok {
							// Encountered a user-requested peripheral type. Initiate the AST
							// depth-first search to identify each of its registers.
							*per = Periph{
								spec:      t,
								ident:     i,
								Registers: []Register{},
							}
							ast.Walk(per.typeVisitor(), per.spec)
							// spew.Dump(periph.Registers)
						}
					}
				}
			}
		}
	}

	// Next, gather all of the const definitions associated with each of our
	// peripheral types.
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			switch d.Tok {
			case token.CONST:
				for _, spec := range d.Specs {
					switch t := spec.(type) {
					case *ast.ValueSpec:
						for i, n := range t.Names {

							_, reg := p.bitFieldRegister(n.Name)
							if reg == nil {
								continue
							}
							bn, bv, bc := reg.parseConst(n.Name)
							if bc == bcErr {
								continue
							}
							bf := reg.Field[bn]

							switch v := t.Values[i].(type) {
							case *ast.BasicLit:
								switch v.Kind {
								case token.INT:
									cv, err := strconv.ParseUint(v.Value, 0, 64)
									if err != nil {
										continue
									}
									switch bc {
									case bcPos:
										bf.pos = uint(cv)
									case bcMsk:
										bf.msk = uint(cv)
									case bcBit:
									case bcVal:
										bf.Enum = append(bf.Enum,
											BitFieldEnum{ident: bv, value: uint(cv)})
									}
								}
							}
							reg.Field[bn] = bf
							// spew.Dump(reg.Field)
						}

					}
				}
			case token.VAR:
			case token.TYPE:
			}
		}
	}

	spew.Dump(p)
}

// bitFieldRegister returns the receiver Package's Periph and its Register
// associated with the given bit field identifier s.
//
// See the godoc comment on type BitField for additional details.
func (p *Package) bitFieldRegister(s string) (per *Periph, reg *Register) {
	if p.Periph != nil {
		for n, h := range p.Periph {
			if strings.HasPrefix(s, n) {
				per = h
				reg = h.bitFieldRegister(s)
			}
		}
	}
	return
}
