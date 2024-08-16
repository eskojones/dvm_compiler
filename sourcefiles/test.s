; test source file

; program entry...
call @main


; includes...
%include "test_functions.s"


; aliases...
~strWelcome       0xf000
~strTest16Int     0xf020
~strTest8Int      0xf040
~strTestByteArray 0xf060
~strNewline       0xf080
~strSuccess       0xf084
~strFailed        0xf090
~i16_test         0xf100
~i8_test1         0xf102
~i8_test2         0xf103
~i8_bytes         0xf104
~result           r0
~arg              r0
~err              r10
~PrintChar        1
~PrintCharAt      2
~PrintStr         3


; data...
.0xf000 "Welcome to the test program.\n\0"
.0xf020 "Testing 16-bit integer: \0"
.0xf040 "Testing 8-bit integer: \0"
.0xf060 "Testing byte array: \0"
.0xf080 "\r\n\0"
.0xf084 "Success!\0"
.0xf090 "Failed!\0"
.0xf100 0x4142
.0xf102 0x41
.0xf103 0x42
.0xf104 [ 0x41, 97, 0x42, 98, 0x43, 99, 0x44, 100, 0x45, 101, 0x0a ]


; code...

:main
    mov $arg, $strWelcome
    int $PrintStr
    call @fnTest16Int
    cmp $err, 1
    jz @end
    call @fnTest8Int
    cmp $err, 1
    jz @end
    call @fnTestByteArray

:end
    push 0x4444         ; push value 0x4444 onto stack
    pop $i16_test       ; pop from stack into address in alias $i16_test
    ld r5, $i16_test    ; load address $i16_test into reg 5
    mov r0, r5          ; move reg 5 into reg 0
    int $PrintChar      ; print the character in the low byte of reg 0
    inc r0              ; increment the value of reg 0
    int $PrintChar      ; print the character in the low byte of reg 0
                        ; ...expected output is "DE" (0x44,0x45)
    hlt

