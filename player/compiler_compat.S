.syntax unified
.cpu arm7tdmi
.arm

.section .text.__aeabi_uidivmod, "ax", %progbits
.global __aeabi_uidivmod
.type __aeabi_uidivmod, %function
__aeabi_uidivmod:
    cmp r1, #0
    beq 2f
    mov r2, #0          @ quotient
    mov r3, #0          @ remainder
    mov r12, #32
1:
    movs r0, r0, lsl #1 @ next numerator bit -> carry
    adc r3, r3, r3      @ remainder = remainder*2 + carry
    mov r2, r2, lsl #1
    cmp r3, r1
    subcs r3, r3, r1
    orrcs r2, r2, #1
    subs r12, r12, #1
    bne 1b
    mov r0, r2
    mov r1, r3
    bx lr
2:
    mov r1, r0
    mov r0, #0
    bx lr
.size __aeabi_uidivmod, .-__aeabi_uidivmod

.section .text.__aeabi_uidiv, "ax", %progbits
.global __aeabi_uidiv
.type __aeabi_uidiv, %function
__aeabi_uidiv:
    b __aeabi_uidivmod
.size __aeabi_uidiv, .-__aeabi_uidiv

.section .text.__aeabi_memclr4, "ax", %progbits
.global __aeabi_memclr4
.global __aeabi_memclr
.type __aeabi_memclr4, %function
.type __aeabi_memclr, %function
__aeabi_memclr4:
__aeabi_memclr:
    cmp r1, #0
    bxeq lr
    mov r2, #0
3:
    strb r2, [r0], #1
    subs r1, r1, #1
    bne 3b
    bx lr
.size __aeabi_memclr4, .-__aeabi_memclr4
