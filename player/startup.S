    .syntax unified
    .cpu arm7tdmi
    .arm

    .section .gba_header, "ax", %progbits
    .global _rom_header
_rom_header:
    .space 0xC0, 0

    .section .text.startup, "ax", %progbits
    .align 2
    .global _start
    .type _start, %function
_start:
    /* Give IRQ mode its own stack, then enter System mode with IRQ enabled. */
    mrs     r0, cpsr
    bic     r1, r0, #0x1F
    orr     r1, r1, #0x12
    orr     r1, r1, #0x80
    msr     cpsr_c, r1
    ldr     sp, =0x03007FA0

    bic     r1, r0, #0x1F
    orr     r1, r1, #0x1F
    bic     r1, r1, #0x80
    msr     cpsr_c, r1
    ldr     sp, =0x03007F00

    bl      main
1:
    b       1b
    .size _start, .-_start
