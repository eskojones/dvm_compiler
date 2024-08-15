%include "file.s"

:test_16bit_integer
    mov r0, $str_testing_16bit_integer
    int 3
    mov r0, 0xf100
    int 2
    mov r0, 0xf101
    int 2
    mov r0, 0xf080
    int 3
    mov r1, 0
    ret

:test_8bit_integer
    mov r0, 0xf040
    int 3
    mov r0, 0xf102
    int 2
    mov r0, 0xf103
    int 2
    mov r0, 0xf080
    int 3
    mov r1, 0
    ret

:test_byte_array
    mov r0, 0xf060                # print "testing byte array"
    int 3                         #
    mov r0, 0xf104                # move the address of byte array into reg 0
    mov r1, r0                    # move reg 0 into reg 1
    add r1, 0x0a                  # add 0x0a to reg 1
    :test_byte_array_loop
        int 2                     # print next character
        inc r0                    # add 1 to reg 0
        cmp r0, r1                # compare reg 0 and reg 1
        jl @test_byte_array_loop  # loop if reg 0 is lessthan reg 1
    mov r0, 0xf080                # print newline string
    int 3                         # 
    mov r1, 0
    ret
