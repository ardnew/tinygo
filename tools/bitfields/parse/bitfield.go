package parse

import "math/bits"

// BitField defines the position, width, mask, and any enumerated values
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
//       integer flags such as "SPEED_0", "SPEED_1", "SPEED_2". This will depend
//       on the quality of your source SVD.
//       Note that in this form, the enumerated values are unshifted.
//
// Depending on the form of the bit field constants, the BitField struct created
// will optionally contain elements in the Enu slice. In the first case, this
// slice will remain nil to help distinguish the two forms.
// All other fields will always be initialized for every bit field.
type BitField struct {
	pos  uint
	msk  uint
	Enum []BitFieldEnum
}

func (b BitField) Len() int   { return bits.OnesCount(b.msk) }
func (b BitField) Msb() int   { return int(b.pos) + b.Len() - 1 }
func (b BitField) Lsb() int   { return int(b.pos) }
func (b BitField) Mask() uint { return b.msk }

type BitFieldEnum struct {
	ident string
	value uint
}

func (b BitFieldEnum) Ident() string { return b.ident }
func (b BitFieldEnum) Value() uint   { return b.value }

type bitFieldConst int

const (
	bcErr bitFieldConst = iota
	bcPos
	bcMsk
	bcBit
	bcVal
)
