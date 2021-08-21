package parse

import "math/bits"

// Field defines the position, width, mask, and any enumerated values
// describing a bit field in a memory-mapped register of some peripheral.
//
// These properties of a bit field are discovered through lexical analysis of
// types and constant values/identifiers defined in Go source code.
//
// Bit field constant identifiers are automatically generated, via TinyGo's
// gen-device-svd, using the identifiers from its respective peripheral type and
// register name as prefixes (see example below).
//
// For a hypothetical example, a device package may define an I2C peripheral
// type "I2C_Type" as a struct with fields for each of its SVD-defined
// registers, including two control registers "CR1" and "CR2".
// The SVD will also define bit fields as fixed-width spans of bits within these
// registers, such as "ENABLE" in "CR1", and "SPEED" in "CR2".
// In this scenario, gen-device-svd -may- generate Go source code containing the
// following definitions:
//
//   type I2C_Type struct {
//	   CR1 volatile.Register32 // 0x0
//	   CR2 volatile.Register32 // 0x4
//     ...
//   }
//
//   const (
//	   I2C_CR1_ENABLE_Pos           = 0x2
//	   I2C_CR1_ENABLE_Msk           = 0x4
//	   I2C_CR1_ENABLE               = 0x4
//     ...
//	   I2C_CR2_SPEED_Pos            = 0x6
//	   I2C_CR2_SPEED_Msk            = 0xc0
//	   I2C_CR2_SPEED_SPEED_LOW      = 0x1
//	   I2C_CR2_SPEED_SPEED_STANDARD = 0x2
//	   I2C_CR2_SPEED_SPEED_HIGH     = 0x3
//   )
//
// Note that there are two different forms of bit field descriptions above:
//
//   1.) The "ENABLE" bit field is only a single bit wide.
//       In this case, depending on the content of the SVD, gen-device-svd will
//       not generate enumerated constants for the two permissible values of the
//       field.
//       Instead, only a single value constant is generated, and is equal to the
//       bit field mask. This value is always pre-shifted into position, and it
//       contains no suffix on its identifier.
//
//   2.) The "SPEED" bit field is 3-bits wide, and it includes enumerated values
//       for each of the SVD-defined constant values allowed in the field.
//       The enumerated value constants may include descriptive names as shown
//       "SPEED_LOW", "SPEED_STANDARD", and "SPEED_HIGH"; or they may be simple
//       ordinal names such as "SPEED_0", "SPEED_1", "SPEED_2". This will depending
//       on the quality of your source SVD.
//       Note that in this form, the enumerated values are unshifted.
//
// Depending on the form of the bit field constants, the Field struct created
// will optionally contain elements in the Enum slice.
// All other struct fields will always be initialized for every bit field.
type Field struct {
	pos  uint
	msk  uint
	reg  *Register
	Enum []FieldEnum
}

func (b Field) Len() int   { return bits.OnesCount(b.msk) }
func (b Field) Msb() int   { return int(b.pos) + b.Len() - 1 }
func (b Field) Lsb() int   { return int(b.pos) }
func (b Field) Pos() int   { return int(b.pos) }
func (b Field) Mask() uint { return b.msk }

// Register gives us a back-reference to the register containing this bit field.
// From it we can deduce contextual things. For example, if a bit field is only
// 4 bits wide, but it belongs to a 32-bit register, we probably want to have a
// 32-bit interface to the 4-bit field for compatibility with TinyGo's general
// Register32.Get/Set methods.
// We cannot know this without having some information about the register it
// belongs to.
func (b Field) Register() *Register { return b.reg }

type FieldEnum struct {
	ident string
	value uint
}

func (b FieldEnum) Ident() string { return b.ident }
func (b FieldEnum) Value() uint   { return b.value }

type fieldConst int

const (
	bcErr fieldConst = iota
	bcPos
	bcMsk
	bcBit
	bcVal
)
