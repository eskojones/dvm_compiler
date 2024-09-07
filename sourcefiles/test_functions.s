%include "file.s"

fnTest16Int:
    mov r0, $strTest16Int
    int $PrintStr
    mov r0, $i16_test
    int $PrintCharAt
    inc r0
    int $PrintCharAt
    mov r0, $strNewline
    int $PrintStr
    mov r1, 0
    ret

fnTest8Int:
    mov r0, $strTest8Int
    int $PrintStr
    mov r0, $i8_test1
    int $PrintCharAt
    mov r0, $i8_test2
    int $PrintCharAt
    mov r0, $strNewline
    int $PrintStr
    mov r1, 0
    ret

fnTestByteArray:
    mov r0, $strTestByteArray     ; print "testing byte array"
    int $PrintStr
    mov r0, $i8_bytes             ; move the address of byte array into reg 0
    mov r1, r0                    ; move reg 0 into reg 1
    add r1, 10                    ; add 10 to reg 1
    .loop:
        int $PrintCharAt          ; print next character
        inc r0                    ; add 1 to reg 0
        cmp r0, r1                ; compare reg 0 and reg 1
        jl @.loop                 ; loop if reg 0 is lessthan reg 1
    mov r0, $strNewline           ; print newline string
    int $PrintStr
    mov r1, 0
    ret
