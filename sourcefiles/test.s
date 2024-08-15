# test source file

.0xf000 "Welcome to the test program.\n\0"
.0xf020 "Testing 16-bit integer: \0"
.0xf040 "Testing 8-bit integer: \0"
.0xf060 "Testing byte array: \0"
.0xf080 "\r\n\0"
.0xf084 "Success!\n\0"
.0xf090 "Failed!\n\0"
.0xf100 0x4142
.0xf102 0x41
.0xf103 0x42
.0xf104 [ 0x41, 97, 0x42, 98, 0x43, 99, 0x44, 100, 0x45, 101, 0x0a ]
.0xf200 1

:start
    mov r0, $str_welcome_msg 
    int 3h
    call @test_16bit_integer
    cmp r1, 1
    jz @failed
    call @test_8bit_integer
    cmp r1, 1
    jz @failed
    call @test_byte_array
    cmp r1, 1
    jz @failed
    mov r0, 0xf084 # success msg
    int 3
    jmp @end

%include "test_functions.s"

:failed
    mov r0, 0xf090
    int 3

:end
    hlt

