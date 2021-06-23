// +build stm32h7x5

package usb

// descHCLKFrequencyHz defines the AHB clock frequency configured by the system.
// It is needed for selecting an appropriate "Turnaround time" (USBTRD), whose
// purpose I personally have a hard time understanding. The following is the
// entirety of documentation I can find in the literature on the subject.
//
// From STM32 HAL source code:
//   The USBTRD is configured [...] depending on AHB frequency used by
//   application. In the low AHB frequency range it is used to stretch enough
//   the USB response time to IN tokens, the USB turnaround time, so to
//   compensate for the longer AHB read access latency to the Data FIFO.
//
// And from the STM32H7 reference manual:
//   The number of PHY clocks that the application programs in this field is
//   added to the full-speed interpacket timeout duration in the core to account
//   for any additional delays introduced by the PHY. This can be required,
//   because the delay introduced by the PHY in generating the line state
//   condition can vary from one PHY to another. The USB standard timeout value
//   for full-speed operation is 16 to 18 (inclusive) bit times. The application
//   must program this field based on the speed of enumeration. The number of
//   bit times added per PHY clock is 0.25 bit times.
//
const descHCLKFrequencyHz = 240000000
