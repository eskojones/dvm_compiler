package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)


type instr struct {
	name   string
	opcode uint8
	size   uint8
}


var instructions = []instr{
	{"nop", 	0x00, 	1},
	{"hlt", 	0xff, 	1},
	{"print", 	0xfe, 	2},
	{"ld", 		0x01, 	4},
	{"st", 		0x02, 	4},
	{"int", 	0x03, 	2},
	{"inti", 	0x04, 	3},
	{"ret", 	0x05, 	1},
	{"mov", 	0x06, 	3},
	{"movi", 	0x07, 	4},
	{"cmp", 	0x08, 	3},
	{"cmpi", 	0x09, 	4},
	{"jmp", 	0x0a, 	2},
	{"jmpi", 	0x0b, 	3},
	{"jz", 		0x0c, 	2},
	{"jzi", 	0x0d, 	3},
	{"jnz", 	0x0e, 	2},
	{"jnzi", 	0x0f, 	3},
	{"jl", 		0x10, 	2},
	{"jli", 	0x11, 	3},
	{"jg", 		0x12, 	2},
	{"jgi", 	0x13, 	3},
	{"inc", 	0x14, 	2},
	{"dec", 	0x15, 	2},
	{"add", 	0x16, 	3},
	{"addi", 	0x17, 	4},
	{"sub", 	0x18, 	3},
	{"subi", 	0x19, 	4},
	{"mul", 	0x1a, 	3},
	{"muli", 	0x1b, 	4},
	{"div",	 	0x1c, 	3},
	{"divi", 	0x1d, 	4},
	{"call", 	0x1e, 	2},
	{"calli", 	0x1f, 	3},
}


type statement struct {
	line_num uint
	address uint16
	source []string
	byte_code [6]byte
	byte_count int
}

var opcode2instr = map[uint8]instr{}
var name2instr = map[string]instr{}


func cleanString (dirty string, comment string) string {
	var clean string = strings.Clone(dirty)
	var idx = strings.Index(clean, comment)
	if idx != -1 {
		clean = clean[0 : idx]
	}
	clean = strings.ReplaceAll(clean, "\t", " ")
	clean = strings.ReplaceAll(clean, ",", " ")
	for strings.Contains(clean, "  ") {
		clean = strings.ReplaceAll(clean, "  ", " ")
	}
	clean = strings.TrimLeft(strings.Trim(clean, "\r "), "\r ")
	return clean
}


func createLabels (statements []*statement) (map[string]uint16, error) {
	labels := map[string]uint16{}
	address := uint16(0)

	for _, s := range statements {
		if s == nil || len(s.source) == 0 || len(s.source[0]) == 0 {
			continue
		}

		instr_name := s.source[0]
		if instr_name[0] == ':' {
			_, label_exists := labels[instr_name]
			if label_exists {
				fmt.Printf("Line %d: Duplicate label (%s)\n", s.line_num, instr_name[1:])
				return labels, errors.New("Duplicate Labels")
			}
			labels[instr_name[1:]] = address
			//fmt.Printf("Line %d: Label %s = 0x%04x\n", instr_name[1:], address)
			continue
		}

		_, exists := name2instr[instr_name]
		if !exists {
			fmt.Printf("Line %d: Invalid instruction (%s)\n", s.line_num, instr_name)
			return labels, errors.New("Invalid Instruction")
		}

		s.byte_count = 0
		isImmediate := false
		for idx, arg_str := range s.source {
			if idx > 0 {
				if arg_str[0] == '$' {
					if isImmediate {
						fmt.Printf("Line %d: More than one argument is an immediate value, this is invalid.\n", s.line_num)
						return labels, errors.New("Multiple Immediate Values")
					}
					s.byte_count++
					isImmediate = true
				} else if arg_str[0] == '@' {
					s.byte_count++
				}
			}
			s.byte_count++
		}
		address += uint16(s.byte_count)
	}

	return labels, nil
}


func parseStatements (statements []*statement, labels map[string]uint16, print_debug bool) error {
	var address uint16 = 0

	for _, s := range statements {
		if s == nil || len(s.source) == 0 || len(s.source[0]) == 0 || s.source[0][0] == ':' {
			continue
		}

		instr_name := s.source[0]
		instruction := name2instr[instr_name]
		s.byte_code[0] = instruction.opcode
		s.byte_count = 0
		arg_int := uint64(0)

		for idx, arg_str := range s.source {
			if idx > 0 {
				if arg_str[0] == 'r' {
					arg_int, _ = strconv.ParseUint(arg_str[1:], 10, 0)
					s.byte_code[s.byte_count] = uint8(arg_int)

				} else if arg_str[0] == '$' {
					//immediate value
					if strings.Index(arg_str, "$0x") == 0 {
						arg_int, _ = strconv.ParseUint(arg_str[3:], 16, 0)
					} else {
						arg_int, _ = strconv.ParseUint(arg_str[1:], 10, 0)
					}
					s.byte_code[0] = instruction.opcode + 1
					s.byte_code[s.byte_count] = uint8(arg_int >> 8)
					s.byte_count++
					s.byte_code[s.byte_count] = uint8(arg_int & 0x00ff)

				} else if arg_str[0] == '@' {
					//label
					arg_int, _ = strconv.ParseUint(fmt.Sprintf("%04x", labels[arg_str[1:]]), 16, 0)
					s.byte_code[0] = instruction.opcode + 1
					s.byte_code[s.byte_count] = uint8(arg_int >> 8)
					s.byte_count++
					s.byte_code[s.byte_count] = uint8(arg_int & 0x00ff)
				}
			}
			s.byte_count++
		}

		s.address = address
		address += uint16(s.byte_count)
		
		if print_debug {
			//debug print the original source side by side with the byte-code
			fmt.Printf("0x%04x ", address)
			sline := ""
			for _, sword := range s.source {
				sline = fmt.Sprintf("%s%s ", sline, sword)
			}
			fmt.Printf("%-20s ", sline)
			for i := 0; i < s.byte_count; i++ {
				fmt.Printf("%02x ", s.byte_code[i])
			}
			fmt.Printf("\n")
		}
	}

	return nil

}



func main() {
	if len(os.Args) != 3 {
		exe, _ := os.Executable()
		fmt.Printf("usage: %s source.file out.file\n", exe)
		return
	}

	source_file := os.Args[1]
	output_file := os.Args[2]

	fmt.Printf("Source File: %s\n", source_file)
	fmt.Printf("Output File: %s\n", output_file)

	// read source file in, convert to string, split to lines[]
	source_bytes, err := os.ReadFile(source_file)
	if err != nil {
		fmt.Printf("Source file not readable!")
		return
	}
	source_string := string(source_bytes[:])
	source_lines := strings.Split(source_string, "\n")

	statements := make([]*statement, 0)

	for line_num, line := range source_lines {
		var clean string = cleanString(line, "#")
		if len(clean) == 0 {
			continue
		}
		line_statement := new(statement)
		line_statement.line_num = uint(line_num + 1)
		line_statement.source = strings.Split(strings.Clone(clean), " ")
		statements = append(statements, line_statement)
	}

	for _, ins := range instructions {
		opcode2instr[ins.opcode] = ins
		name2instr[ins.name] = ins
	}

	labels, _ := createLabels(statements)
	err = parseStatements(statements, labels, true)
	if err != nil {
		return
	}

	bytes_out := make([]byte, 0)
	for _, s := range statements {
		for i := 0; i < s.byte_count; i++ {
			bytes_out = append(bytes_out, s.byte_code[i])
		}
	}

	err = os.WriteFile(output_file, bytes_out, 0666)
	if err != nil {
		fmt.Printf("Failed to write output bytecode!\n")
	}
}
