MEMORY
{
  FLASH   (rx) : ORIGIN = 0x08100000, LENGTH = 1024K
  SRAMD2 (rwx) : ORIGIN = 0x10000000, LENGTH =  288K /* Cortex-M4 domain */
  SRAMD3 (rwx) : ORIGIN = 0x38000000, LENGTH =   64K /* Shared domain */
}

ENTRY(Reset_Handler);

_stack_size = 4K;

SECTIONS
{
  /* Program code and read-only data goes to FLASH. */
  .text :
  {
    _svectors = ABSOLUTE(.);
    KEEP(*(.isr_vector))
    . = ALIGN(32);
    _evectors = .; /* TODO: copy to SRAM? */
    *(.text.Reset_Handler);
    . = ALIGN(32);
    _stext = .;
    *(.text*)
    *(.rodata*)
    *(.constdata*)
    /* Cortex-M4 core does not have access to the TCM regions. Objects in these
        sections are placed alongside ordinary code. */
    *(.fastrun);
    *(.isr);
    . = ALIGN(8);
    _etext = .;

  } > FLASH

  .tinygo_stacksizes :
  {
    *(.tinygo_stacksizes)
  } > FLASH

  .stack (NOLOAD) : {

    . = ALIGN(16);
    . += _stack_size;
    _stack_top = .;

  } > SRAMD2

  .heap (NOLOAD) : {

    . = ALIGN(16);
    _heap_start = .;
    /* Use all remaining SRAM for heap */
    . += ORIGIN(DTCM) + LENGTH(DTCM) -  _heap_start;
    _heap_end = .;

  } > SRAMD2

  .data : {

    . = ALIGN(16);
    _sdata = .;
    *(.data*);
    . = ALIGN(16);
    _edata = .;

  } > SRAMD2 AT > FLASH

  .bss : {

    . = ALIGN(16);
    _sbss = .;
    *(.bss*);
    *(COMMON);
    . = ALIGN(16);
    _ebss = .;

  } > SRAMD2

  /DISCARD/ :
  {
    *(.ARM.exidx*) /* causes spurious linker errors */
  }
}

_sidata = LOADADDR(.data);

/* For the memory allocator. */
_globals_start = _sdata;
_globals_end = _ebss;
