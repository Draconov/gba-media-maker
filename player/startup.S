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
    mov     r0, #0x03000000
    add     sp, r0, #0x7F00
    add     sp, sp, #0xC0
    bl      main
1:
    b       1b
    .size _start, .-_start
