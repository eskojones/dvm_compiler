# test source file


# program entry...
call @main


# includes...
%include "test_functions.s"


# aliases...
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
~error            r10
~PrintChar        1
~PrintCharAt      2
~PrintStr         3


# data...
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


# code...

:main
    mov $arg, $strWelcome
    int $PrintStr
    call @fnTest16Int
    cmp $error, 1
    jz @failed
    call @fnTest8Int
    cmp $error, 1
    jz @failed
    call @fnTestByteArray
    cmp $error, 1
    jz @failed
    mov $arg, $strSuccess
    int $PrintStr
    jmp @end

:failed
    mov $arg, $strFailed
    int $PrintStr

:end
    hlt

