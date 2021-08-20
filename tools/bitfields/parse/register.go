package parse

import "strings"

type Register struct {
	ident   string
	prefix  string
	size    int
	offset  int
	comment string
	Field   map[string]BitField
}

func (r *Register) Ident() string   { return r.ident }
func (r *Register) Prefix() string  { return r.prefix }
func (r *Register) Size() int       { return r.size }
func (r *Register) Offset() int     { return r.offset }
func (r *Register) Comment() string { return r.comment }

func (r *Register) parseConst(s string) (fld, val string, typ bitFieldConst) {

	name := strings.TrimPrefix(s, r.prefix)

	var ok bool
	var id string

	if ok, id = r.match(name, bcPos); ok {
		fld, typ = id, bcPos
	} else if ok, id = r.match(name, bcMsk); ok {
		fld, typ = id, bcMsk
	} else if ok, id = r.match(name, bcBit); ok {
		fld, typ = id, bcBit
	} else if ok, id = r.match(name, bcVal); ok {
		fld, val, typ = id, strings.TrimPrefix(name, id+"_"), bcVal
	} else {
		typ = bcErr
	}

	return
}

func (r *Register) match(s string, c bitFieldConst) (bool, string) {
	switch c {
	case bcPos:
		// TrimSuffix already tests if suffix exists, so don't duplicate effort.
		n := strings.TrimSuffix(s, "_Pos")
		// len(n) < len(s) iff HasSuffix(s, "_Pos")
		return len(n) < len(s), n

	case bcMsk:
		// See case bcPos above for reasoning
		n := strings.TrimSuffix(s, "_Msk")
		return len(n) < len(s), n

	case bcBit:
		// The bcBit constant identifiers consist only of the field name without any
		// suffix. In which case, we should already have a BitField defined in our
		// receiver's Field map (for the bcPos and bcMsk constants).
		//   TODO: Verify bcPos and bcMsk constants are defined lexically prior to
		//         all bcBit and bcVal constants. I believe this is the case with
		//         our TinyGo gen-device-svd generator, but I haven't verified it.
		_, ok := r.Field[s]
		return ok, s

	case bcVal:
		// Check each existing BitField on this register for the longest identifier
		// that is a prefix of our input string.
		//   TODO: Like the bcBit note above, this logic depends on the enumerated
		//         value constants (bcVal) to be defined after the bcPos and bcMsk
		//         identifiers. Otherwise, we can't reliably extract the field name
		//         or, thus, the enumerated value suffix from the identifier. Need
		//         to verify this is always true or devise another plan of attack.
		var n string
		for f := range r.Field {
			g := strings.TrimPrefix(s, f)
			if len(g) < len(s) && len(n) < len(f) {
				// The current iterated BitField's identifier is a prefix of the given
				// const identifier, and it is the longest such prefix found so far.
				n = f
			}
		}
		return n != "", n

	}

	return false, ""
}
