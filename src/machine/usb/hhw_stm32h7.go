// +build stm32h7

package usb

// Implementation of USB host controller hardware abstraction (hhw) for STM32H7.

//   +-- [ NOTE ] --------------------------------------------------------+
//   |                                                                    |
//   |  Often you will see suffixes "HS" and "FS" when referring to core  |
//   |  registers and bitmasks. These are automatically-generated naming  |
//   |  conventions used to distinguish the two USB cores and are not     |
//   |  actually dependent on configured bus speed.                       |
//   |                                                                    |
//   |  "HS" refers to USB PHY/core 1, since it is the only core that is  |
//   |  capable of high-speed mode. "FS" refers to USB PHY/core 2.        |
//   |                                                                    |
//   +--------------------------------------------------------------------+
