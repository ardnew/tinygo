// Hand created file. DO NOT DELETE.
// Type definitions, fields, and constants associated with FlexPWM peripherals
// of the NXP MIMXRT1062.

//go:build nxp && mimxrt1062

package nxp

import (
	"runtime/volatile"
	"unsafe"
)

type FlexPWM struct {
	*FLEXPWM_Type // structure containing all peripheral registers.

	id FlexPWMID
}

const (
	NumFlexPWM = 4 // number of FlexPWM peripherals of iMXRT1062
)

type FlexPWMID uint32

// Enumerated constant values for each FlexPWM hardware peripheral. Each FlexPWM
// instance can be referred to by its corresponding FlexPWMID number.
const (
	FPWM1 FlexPWMID = 1 << iota
	FPWM2
	FPWM3
	FPWM4
)

func (id FlexPWMID) FlexPWM() *FlexPWM {
	switch id {
	case FPWM1:
		return &FlexPWM1
	case FPWM2:
		return &FlexPWM2
	case FPWM3:
		return &FlexPWM3
	case FPWM4:
		return &FlexPWM4
	}
	return nil
}

var (
	FlexPWM1 = FlexPWM{
		FLEXPWM_Type: FLEXPWM1,
		id:           FPWM1,
	}
	FlexPWM2 = FlexPWM{
		FLEXPWM_Type: FLEXPWM2,
		id:           FPWM2,
	}
	FlexPWM3 = FlexPWM{
		FLEXPWM_Type: FLEXPWM3,
		id:           FPWM3,
	}
	FlexPWM4 = FlexPWM{
		FLEXPWM_Type: FLEXPWM4,
		id:           FPWM4,
	}
)

func (f *FlexPWM) ID() FlexPWMID { return f.id }

type FlexPWMChan uint8

const (
	FPWMX FlexPWMChan = 1 << iota
	FPWMA
	FPWMB
)

type FlexPWMPin struct {
	Bus  *FlexPWM    // FlexPWM peripheral instance (FlexPWM1-4)
	Sub  uint8       // FlexPWM submodule number (0-3)
	Chan FlexPWMChan // FlexPWM signal channel (FPWM[A], [B], or [X])
	Mux  uint8       // IOMUXC pin alternate function number for FlexPWM (0-9)
}

func (p FlexPWMPin) DMAChannel() (read, write DMAChannelID) {
	switch p.Bus.ID() {
	case FPWM1:
		switch p.Sub {
		case 0:
			return dmaSourceFlexPWM1Read0, dmaSourceFlexPWM1Write0
		case 1:
			return dmaSourceFlexPWM1Read1, dmaSourceFlexPWM1Write1
		case 2:
			return dmaSourceFlexPWM1Read2, dmaSourceFlexPWM1Write2
		case 3:
			return dmaSourceFlexPWM1Read3, dmaSourceFlexPWM1Write3
		}
	case FPWM2:
		switch p.Sub {
		case 0:
			return dmaSourceFlexPWM2Read0, dmaSourceFlexPWM2Write0
		case 1:
			return dmaSourceFlexPWM2Read1, dmaSourceFlexPWM2Write1
		case 2:
			return dmaSourceFlexPWM2Read2, dmaSourceFlexPWM2Write2
		case 3:
			return dmaSourceFlexPWM2Read3, dmaSourceFlexPWM2Write3
		}
	case FPWM3:
		switch p.Sub {
		case 0:
			return dmaSourceFlexPWM3Read0, dmaSourceFlexPWM3Write0
		case 1:
			return dmaSourceFlexPWM3Read1, dmaSourceFlexPWM3Write1
		case 2:
			return dmaSourceFlexPWM3Read2, dmaSourceFlexPWM3Write2
		case 3:
			return dmaSourceFlexPWM3Read3, dmaSourceFlexPWM3Write3
		}
	case FPWM4:
		switch p.Sub {
		case 0:
			return dmaSourceFlexPWM4Read0, dmaSourceFlexPWM4Write0
		case 1:
			return dmaSourceFlexPWM4Read1, dmaSourceFlexPWM4Write1
		case 2:
			return dmaSourceFlexPWM4Read2, dmaSourceFlexPWM4Write2
		case 3:
			return dmaSourceFlexPWM4Read3, dmaSourceFlexPWM4Write3
		}
	}
	return
}

type FLEXPWM_Type struct {
	SM [4]struct { //                   SM0    SM1    SM2    SM3
		CNT       volatile.Register16 //  0x00   0x60   0xC0   0x120
		INIT      volatile.Register16 //  0x02   0x62   0xC2   0x122
		CTRL2     volatile.Register16 //  0x04   0x64   0xC4   0x124
		CTRL      volatile.Register16 //  0x06   0x66   0xC6   0x126
		_         [2]byte             //  0x08   0x68   0xC8   0x128
		VAL0      volatile.Register16 //  0x0A   0x6A   0xCA   0x12A
		FRACVAL1  volatile.Register16 //  0x0C   0x6C   0xCC   0x12C
		VAL1      volatile.Register16 //  0x0E   0x6E   0xCE   0x12E
		FRACVAL2  volatile.Register16 //  0x10   0x70   0xD0   0x130
		VAL2      volatile.Register16 //  0x12   0x72   0xD2   0x132
		FRACVAL3  volatile.Register16 //  0x14   0x74   0xD4   0x134
		VAL3      volatile.Register16 //  0x16   0x76   0xD6   0x136
		FRACVAL4  volatile.Register16 //  0x18   0x78   0xD8   0x138
		VAL4      volatile.Register16 //  0x1A   0x7A   0xDA   0x13A
		FRACVAL5  volatile.Register16 //  0x1C   0x7C   0xDC   0x13C
		VAL5      volatile.Register16 //  0x1E   0x7E   0xDE   0x13E
		FRCTRL    volatile.Register16 //  0x20   0x80   0xE0   0x140
		OCTRL     volatile.Register16 //  0x22   0x82   0xE2   0x142
		STS       volatile.Register16 //  0x24   0x84   0xE4   0x144
		INTEN     volatile.Register16 //  0x26   0x86   0xE6   0x146
		DMAEN     volatile.Register16 //  0x28   0x88   0xE8   0x148
		TCTRL     volatile.Register16 //  0x2A   0x8A   0xEA   0x14A
		DISMAP0   volatile.Register16 //  0x2C   0x8C   0xEC   0x14C
		DISMAP1   volatile.Register16 //  0x2E   0x8E   0xEE   0x14E
		DTCNT0    volatile.Register16 //  0x30   0x90   0xF0   0x150
		DTCNT1    volatile.Register16 //  0x32   0x92   0xF2   0x152
		CAPTCTRLA volatile.Register16 //  0x34   0x94   0xF4   0x154
		CAPTCOMPA volatile.Register16 //  0x36   0x96   0xF6   0x156
		CAPTCTRLB volatile.Register16 //  0x38   0x98   0xF8   0x158
		CAPTCOMPB volatile.Register16 //  0x3A   0x9A   0xFA   0x15A
		CAPTCTRLX volatile.Register16 //  0x3C   0x9C   0xFC   0x15C
		CAPTCOMPX volatile.Register16 //  0x3E   0x9E   0xFE   0x15E
		CVAL0     volatile.Register16 //  0x40   0xA0   0x100  0x160
		CVAL0CYC  volatile.Register16 //  0x42   0xA2   0x102  0x162
		CVAL1     volatile.Register16 //  0x44   0xA4   0x104  0x164
		CVAL1CYC  volatile.Register16 //  0x46   0xA6   0x106  0x166
		CVAL2     volatile.Register16 //  0x48   0xA8   0x108  0x168
		CVAL2CYC  volatile.Register16 //  0x4A   0xAA   0x10A  0x16A
		CVAL3     volatile.Register16 //  0x4C   0xAC   0x10C  0x16C
		CVAL3CYC  volatile.Register16 //  0x4E   0xAE   0x10E  0x16E
		CVAL4     volatile.Register16 //  0x50   0xB0   0x110  0x170
		CVAL4CYC  volatile.Register16 //  0x52   0xB2   0x112  0x172
		CVAL5     volatile.Register16 //  0x54   0xB4   0x114  0x174
		CVAL5CYC  volatile.Register16 //  0x56   0xB6   0x116  0x176
		_         [2]byte             //  0x58   0xB8   0x118  0x178
		_         [2]byte             //  0x5A   0xBA   0x11A  0x17A
		_         [2]byte             //  0x5C   0xBC   0x11C  0x17C
		_         [2]byte             //  0x5E   0xBE   0x11E  0x17E
	}
	OUTEN    volatile.Register16 // 0x180
	MASK     volatile.Register16 // 0x182
	SWCOUT   volatile.Register16 // 0x184
	DTSRCSEL volatile.Register16 // 0x186
	MCTRL    volatile.Register16 // 0x188
	MCTRL2   volatile.Register16 // 0x18A
	FCTRL0   volatile.Register16 // 0x18C
	FSTS0    volatile.Register16 // 0x18E
	FFILT0   volatile.Register16 // 0x190
	FTST0    volatile.Register16 // 0x192
	FCTRL20  volatile.Register16 // 0x194
}

var (
	FLEXPWM1 = (*FLEXPWM_Type)(unsafe.Pointer(uintptr(0x403DC000)))
	FLEXPWM2 = (*FLEXPWM_Type)(unsafe.Pointer(uintptr(0x403E0000)))
	FLEXPWM3 = (*FLEXPWM_Type)(unsafe.Pointer(uintptr(0x403E4000)))
	FLEXPWM4 = (*FLEXPWM_Type)(unsafe.Pointer(uintptr(0x403E8000)))
)
