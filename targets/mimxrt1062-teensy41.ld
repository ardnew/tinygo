MEMORY
{
  ITCM      (rwx): ORIGIN = 0x00000000, LENGTH = 0x00080000 /*   512K */
  DTCM      (rwx): ORIGIN = 0x20000000, LENGTH = 0x00080000 /*   512K */
  RAM       (rwx): ORIGIN = 0x20200000, LENGTH = 0x00080000 /*   512K */
  FLASH      (rx): ORIGIN = 0x60000000, LENGTH = 0x007C0000 /*  7936K */
  ERAM      (rwx): ORIGIN = 0x70000000, LENGTH = 0x01000000 /* 16384K */
}

ENTRY(Reset_Handler);

_stack_size = 4K;

SECTIONS
{
  .text : {

    FILL(0xFF);

    /* place flash config at beginning of flash device */
    KEEP(*(.flash_config));

    /* IVT must be located at +4 Kbyte offset from base address of flash. */
    . = ORIGIN(FLASH) + 0x1000;
    KEEP(*(.ivt));

    . = ORIGIN(FLASH) + 0x1020;
    KEEP(*(.boot_data));

    . = ORIGIN(FLASH) + 0x2000;
    _svectors = ABSOLUTE(.);
    KEEP(*(.isr_vector));
    . = ALIGN(8);

    . += 32;
    _stext = .;
    *(.text.Reset_Handler);
    *(.text*);
    . = ALIGN(16);
    _etext = .;

  } > FLASH

  .tinygo_stacksizes : {

    *(.tinygo_stacksizes);
    . = ALIGN(16);

  } > FLASH

  .text.padding (NOLOAD) : {

    . = ALIGN(32768);

  } > ITCM

  .stack (NOLOAD) : {

    . = ALIGN(16);
    . += _stack_size;
    _stack_top = .;

  } > DTCM

  .data : {

    FILL(0xFF);

    _sdata = .;
    *(.rodata* .constdata*);
    *(.data*);
    . = ALIGN(16);
    _edata = .;

  } > DTCM AT > FLASH

  .bss : {

    _sbss = .;
    *(.bss*);
    *(COMMON);
    . = ALIGN(32);
    _ebss = .;

  } > DTCM

  .bss.dma (NOLOAD) : {

    *(.dmabuffer);
    . = ALIGN(32);

  } > RAM

  .bss.extram (NOLOAD) : {

    *(.externalram);

  } > RAM

  /DISCARD/ : {

    *(.ARM.exidx*); /* causes spurious 'undefined reference' errors */

  }

  _sidata = LOADADDR(.data);

  _heap_start = ADDR(.bss.dma) + SIZEOF(.bss.dma);
  _heap_end = ORIGIN(RAM) + LENGTH(RAM);

  _globals_start = _sdata;
  _globals_end = _ebss;

  _image_size = SIZEOF(.text) + SIZEOF(.tinygo_stacksizes) + SIZEOF(.data);

  /* TODO: link .text to ITCM */
  _itcm_blocks = (0 + 0x7FFF) >> 15;
  _flexram_cfg = 0xAAAAAAAA | ((1 << (_itcm_blocks * 2)) - 1);
}
