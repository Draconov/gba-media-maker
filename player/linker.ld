ENTRY(_start)

MEMORY
{
    ROM   (rx)  : ORIGIN = 0x08000000, LENGTH = 32M
    EWRAM (rwx) : ORIGIN = 0x02000000, LENGTH = 256K
    IWRAM (rwx) : ORIGIN = 0x03000000, LENGTH = 32K
}

SECTIONS
{
    .gba_header ORIGIN(ROM) :
    {
        KEEP(*(.gba_header))
    } > ROM

    .text : ALIGN(4)
    {
        KEEP(*(.text.startup))
        *(.text*)
        *(.rodata*)
    } > ROM

    .metadata 0x08007FC0 : ALIGN(4)
    {
        KEEP(*(.metadata))
    } > ROM

    .data : ALIGN(4)
    {
        *(.data*)
    } > EWRAM AT > ROM

    .bss (NOLOAD) : ALIGN(4)
    {
        __bss_start__ = .;
        *(.bss*)
        *(COMMON)
        __bss_end__ = .;
    } > EWRAM

    /DISCARD/ :
    {
        *(.comment*)
        *(.ARM.attributes*)
        *(.note*)
        *(.ARM.exidx*)
        *(.ARM.extab*)
    }
}
