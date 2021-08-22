package parse

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

type Periph struct {
	spec     *ast.TypeSpec
	ident    string
	pkg      *Package
	Register []Register

	// offset accumulates the size of each register field in a Periph struct
	// visited via depth-first AST traversal.
	offset int
}

func (p *Periph) Spec() *ast.TypeSpec { return p.spec }
func (p *Periph) Ident() string       { return p.ident }
func (p *Periph) Name() string        { return periphIdent(p.ident) }
func (p *Periph) Package() *Package   { return p.pkg }

func (p *Periph) addRegister(f *ast.Field) (added int) {
	if f != nil && f.Names != nil {
		// If multiple Names of the same type are declared in a single Field
		// specification, then add one register for each name and use the same type
		// repeatedly with each of those registers.
		//
		// For example, in `type S struct { foo, bar T }`, two registers would be
		// added `foo T` and `bar T` to peripheral type T.
		for _, name := range f.Names {
			// Only keep exported fields, but be sure to adjust offset for the size of
			// all fields, whether we keep them or not.
			size := p.structFieldSize(f.Type)
			if name.IsExported() {
				p.Register = append(p.Register,
					Register{
						ident:   name.Name,
						prefix:  periphIdent(p.ident, name.Name) + "_",
						isBlank: false,
						size:    size,
						offset:  p.offset,
						comment: commentText(f.Comment),
						periph:  p,
						Field:   map[string]Field{},
					})
				added += 1
			} else if name.Name == "_" {
				p.Register = append(p.Register,
					Register{
						ident:   "_",
						prefix:  "_",
						isBlank: true,
						size:    size,
						offset:  p.offset,
						comment: commentText(f.Comment),
						periph:  p,
						Field:   map[string]Field{},
					})
				added += 1
			}
			p.offset += size
		}
	}
	return
}

// fieldRegister returns the receiver Periph's Register whose Prefix field is a
// prefix of the given bit field identifier s, or nil if no such Register was
// found.
//
// See the godoc comment on type Field for additional details.
func (p *Periph) fieldRegister(s string) *Register {
	if p.Register != nil {
		for _, r := range p.Register {
			if strings.HasPrefix(s, r.prefix) {
				return &r
			}
		}
	}
	return nil
}

// structFieldSize returns the size (in bytes) of the data type of a struct
// field expressed by the given ast.Expr node.
func (p *Periph) structFieldSize(e ast.Expr) int {
	if e == nil {
		return 0
	}

	switch t := e.(type) {
	case *ast.SelectorExpr:
		switch x := t.X.(type) {
		case *ast.Ident:
			switch x.Name {
			case "volatile":
				switch t.Sel.Name {
				case "Register8":
					return 1
				case "Register16":
					return 2
				case "Register32":
					return 4
				case "Register64":
					return 8
				}
			}
		}
		// Unrecognized field type expresion. You should probably fix me if we
		// ever reach here! Add more packaged types emitted by gen-device-svd.
		return 0

	case *ast.ArrayType:
		if t.Len == nil {
			// Slice type, cannot determine field size
			return 0
		}
		var count int
		switch l := t.Len.(type) {
		case *ast.BasicLit:
			switch l.Kind {
			case token.INT:
				c, _ := strconv.ParseUint(l.Value, 0, 64)
				count = int(c)
			}
		}
		if count == 0 {
			// We couldn't determine array size (not a literal int expression).
			// Not supported.
			return 0
		}
		switch e := t.Elt.(type) {
		case *ast.Ident:
			switch e.Name {
			case "int8", "uint8", "byte":
				return count
			case "int16", "uint16":
				return 2 * count
			case "int32", "uint32", "float32", "rune":
				return 4 * count
			case "int64", "uint64", "float64", "complex64":
				return 8 * count
			case "complex128":
				return 16 * count
			}
		}
		// Unhandled array element type. You should probably fix me if we ever
		// reach here! Add more array types emitted by gen-device-svd.
		return 0

	default:
		return 0
	}
}

// Type definitions for methods that implement ast.Visitor for various kinds of
// declaration blocks found in a single Go source file.
//
// It's awkward using a single (*Periph).Visit method serving as proxy to other
// specific "Visit" methods (e.g., "TypeVisit", "ConstVisit", etc.). This sole
// (*Periph).Visit performs no "Visit" logic itself, but rather only selects
// the appropriate method to call, and then forwards that method's results back
// to the original Periph receiver. It has to do this for every Node visited,
// so its also rather expensive.
//
// Instead, we use methods that take no arguments, and return a reference to
// the Periph receiver they are bound to. These methods each have a differently
// named type according to the purpose of the "Visit" method they implement.
// Since the methods themselves are the receivers of each (*Method).Visit
// method, we can simply call the (*Method).Visit receiver to get a reference to
// the Periph instance being scanned. These are the various periph*Visitor types
// defined below.
type (
	// periphTypeVisitor defines a type that implements ast.Visitor for visiting
	// Nodes of type TypeSpec.
	periphTypeVisitor func() *Periph
)

// self is used as a closure over its receiver from the context of each of the
// *Visitor methods defined on type Periph.
func (p *Periph) self() *Periph                  { return p }
func (p *Periph) typeVisitor() periphTypeVisitor { return p.self }

// Visit will traverse a given Go package AST rooted at a given "type ()" block
// of declarations.
func (p periphTypeVisitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}
	switch t := n.(type) {
	case *ast.StructType:
		if t.Fields != nil && t.Fields.List != nil {
			for _, field := range t.Fields.List {
				_ = p().addRegister(field)
			}
			return nil
		}
	}
	return p
}

// periphIdent returns the base name of a peripheral type emitted from the
// TinyGo SVD-generated device descriptor. The base name is used to associate
// types with their respective const bit field identifiers defined in the same
// source file.
func periphIdent(id string, r ...string) string {
	if r != nil && len(r) > 0 {
		return strings.Replace(id, "Type", r[0], 1)
	}
	return strings.TrimSuffix(id, "_Type")
}

func commentText(g *ast.CommentGroup) string {
	if g == nil {
		return ""
	}
	return strings.TrimSpace(g.Text())
}
