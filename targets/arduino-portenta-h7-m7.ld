MEMORY
{
  FLASH   (rx) : ORIGIN = 0x08000000, LENGTH = 1024K
  ITCM   (rwx) : ORIGIN = 0x00000000, LENGTH =   64K /* VTOR, ISRs */
  DTCM   (rwx) : ORIGIN = 0x20000000, LENGTH =  128K /* Stack + Heap */
  SRAMD1 (rwx) : ORIGIN = 0x24000000, LENGTH =  512K
  SRAMD2 (rwx) : ORIGIN = 0x30000000, LENGTH =  288K
  SRAMD3 (rwx) : ORIGIN = 0x38000000, LENGTH =   64K
}

ENTRY(Reset_Handler);

_stack_size = 4K;

SECTIONS
{
  /* Program code and read-only data goes to FLASH. */
  .text :
  {
    _svectors = ABSOLUTE(.);
    /* Copied to start of ITCM during init */
    KEEP(*(.isr_vector));
    . = ALIGN(32);
    _evectors = ABSOLUTE(.);
    *(.text.Reset_Handler);
    . = ALIGN(32);
    _stext = .;
    *(.text*);
    *(.rodata*);
    *(.constdata*);
    . = ALIGN(8);
    _etext = .;

  } > FLASH

  .tinygo_stacksizes :
  {

    *(.tinygo_stacksizes)

  } > FLASH

  .stack (NOLOAD) : {

    . = ALIGN(4);
    . += _stack_size;
    _stack_top = .;

  } > DTCM

  .heap (NOLOAD) : {

    . = ALIGN(4);
    _heap_start = .;
    /* Use all remaining DTCM for heap */
    . += LENGTH(DTCM) -  _stack_size;
    _heap_end = .;

  } > DTCM

  .critical (NOLOAD) : {

    /* Reserve space for VTOR at start of ITCM */
    _svtor = ABSOLUTE(.);
    . += _evectors - _svectors;
    . = ALIGN(32);
    /* User-defined ISRs placed in ITCM */
    *(.fastrun);
    *(.isr);
    . = ALIGN(32);

  } > ITCM

  .data : {

    . = ALIGN(16);
    _sdata = .;
    *(.data*);
    . = ALIGN(16);
    _edata = .;

  } > SRAMD1 AT > FLASH

  .bss : {

    . = ALIGN(16);
    _sbss = .;
    *(.bss*);
    *(COMMON);
    . = ALIGN(16);
    _ebss = .;

  } > SRAMD1

  /DISCARD/ : {

    *(.ARM.exidx*) /* causes spurious linker errors */

  }
}

_sidata = LOADADDR(.data);

/* For the memory allocator. */
_globals_start = _sdata;
_globals_end = _ebss;
