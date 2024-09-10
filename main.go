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
}

var instructions = []instr{
	{"nop", 0x00},
	{"ldi", 0x01},
	{"sti", 0x02},
	{"int", 0x03},
	{"inti", 0x04},
	{"ret", 0x05},
	{"mov", 0x06},
	{"movi", 0x07},
	{"cmp", 0x08},
	{"cmpi", 0x09},
	{"jmp", 0x0a},
	{"jmpi", 0x0b},
	{"jz", 0x0c},
	{"jzi", 0x0d},
	{"jnz", 0x0e},
	{"jnzi", 0x0f},
	{"jl", 0x10},
	{"jli", 0x11},
	{"jg", 0x12},
	{"jgi", 0x13},
	{"inc", 0x14},
	{"dec", 0x15},
	{"add", 0x16},
	{"addi", 0x17},
	{"sub", 0x18},
	{"subi", 0x19},
	{"mul", 0x1a},
	{"muli", 0x1b},
	{"div", 0x1c},
	{"divi", 0x1d},
	{"call", 0x1e},
	{"calli", 0x1f},
	{"push", 0x20},
	{"pushi", 0x21},
	{"pop", 0x22},
	{"popi", 0x23},
	// mostly unused/debug instructions below
	{"ld", 0xfc},
	{"st", 0xfd},
	{"print", 0xfe},
	{"hlt", 0xff},
}

type statement struct {
	line_num   uint
    label      string
	address    uint16
	source     []string
	byte_code  [4]byte
	byte_count int
}

var opcode2instr = map[uint8]instr{}
var name2instr = map[string]instr{}
var currentProc = ""
    
var commentChar = ";"

func cleanString(dirty string, comment string) string {
	var clean = strings.Clone(dirty)
	var idx = strings.Index(clean, comment)
	if idx != -1 {
		clean = clean[0:idx]
	}
	clean = strings.ReplaceAll(clean, "\t", " ")
	clean = strings.ReplaceAll(clean, ",", " ")
	for strings.Contains(clean, "  ") {
		clean = strings.ReplaceAll(clean, "  ", " ")
	}
	clean = strings.TrimLeft(strings.Trim(clean, "\r "), "\r ")
	return clean
}

func parseAliases(statements []*statement) error {
	aliases := map[string]string{}
	for _, s := range statements {
		if s == nil || len(s.source) == 0 || len(s.source[0]) == 0 || s.source[0][0] == ':' || s.source[0][0] == '.' {
			continue
		}

		for idx, arg_str := range s.source {
			if idx == 0 && arg_str[0] == '~' {
				// make aliases
				aliases[arg_str[1:]] = s.source[idx+1]
				fmt.Printf("alias '%s' = '%s'\n", arg_str[1:], s.source[idx+1])
				continue
			}
		}
	}

	for _, s := range statements {
		if s == nil || len(s.source) == 0 || len(s.source[0]) == 0 || s.source[0][0] == ':' || s.source[0][0] == '.' {
			continue
		}

		for idx, arg_str := range s.source {
			if idx > 0 && arg_str[0] == '$' {
				// apply aliases
				alias_name := arg_str[1:]
				alias_value, alias_exists := aliases[alias_name]
				if !alias_exists {
					fmt.Printf("Line %d: Invalid alias referenced (%s)\n", s.line_num, alias_name)
					return errors.New("invalid alias referenced")
				}
				s.source[idx] = alias_value
			}
		}
	}

	return nil
}

func createLabels(statements []*statement) (map[string]uint16, error) {
	labels := map[string]uint16{}
	address := uint16(0)

	for _, s := range statements {
		if s == nil || len(s.source) == 0 || len(s.source[0]) == 0 {
			continue
		}

		instr_name := s.source[0]
		if instr_name[len(instr_name)-1] == ':' {
            label := instr_name[:len(instr_name) - 1]
            if instr_name[0] == '.' {
                //sub-label
                label = fmt.Sprintf("%s%s", currentProc, label)
            } else {
                currentProc = label
            }
			_, label_exists := labels[label]
			if label_exists {
				fmt.Printf("Line %d: Duplicate label (%s)\n", s.line_num, label)
				return labels, errors.New("duplicate labels")
			}
			labels[label] = address
			fmt.Printf("Line %d: Label %s = 0x%04x\n", s.line_num, label, address)
			continue
		}

        s.label = currentProc
        
        if instr_name[0] == '~' || strings.ContainsAny(instr_name[:2], "1234567890") {
			// data or alias (ignore this until parse)
			continue
		}

		_, exists := name2instr[instr_name]
		if !exists {
			fmt.Printf("Line %d: Invalid instruction (%s)\n", s.line_num, instr_name)
			return labels, errors.New("invalid instruction")
		}

		s.byte_count = 0
		isImmediate := false

		if len(s.source) > 3 {
			fmt.Printf("Line %d: Too many arguments for statement \"%s\".\n", s.line_num, instr_name)
			return labels, errors.New("too many arguments")
		}

		for idx, arg_str := range s.source {
			if idx > 0 {
				if strings.ContainsAny(arg_str[0:1], "0123456789") || arg_str[0] == '@' {
					if isImmediate {
						fmt.Printf("Line %d: Multiple immediate values, this is invalid.\n", s.line_num)
						return labels, errors.New("multiple immediate values")
					}
					s.byte_count++
					isImmediate = true
				}
			}

			s.byte_count++
		}
		address += 4
	}

	return labels, nil
}

func parseIntData(data string) (uint8, uint16) {
	var intval uint64
	if strings.Index(data, "0x") == 0 {
		intval, _ = strconv.ParseUint(data[2:], 16, 0)
    } else if data[len(data)-1] == 'h' {
        intval, _ = strconv.ParseUint(data[0:len(data)-1], 16, 0)
    } else {
		intval, _ = strconv.ParseUint(data, 10, 0)
	}
	byte_count := 1
	if intval > 0xff {
		byte_count = 2
	}
	return uint8(byte_count), uint16(intval)
}

func parseData(data string) []byte {
	bytes := make([]byte, 0)
	var str = strings.Trim(strings.TrimLeft(data, " "), " ")
	if str[0] == '[' && str[len(str)-1] == ']' {
		// [ 0x45, 123 ]
		var strs = strings.Split(str[1:len(str)-1], " ")
		for _, i := range strs {
			if len(i) == 0 {
				continue
			}
			num_bytes, val := parseIntData(i)
			if num_bytes == 2 {
				bytes = append(bytes, uint8(val>>8))
			}
			bytes = append(bytes, uint8(val&0xff))
		}

	} else if (strings.Index(str, "0x") == 0 && len(str) <= 6) || (str[0] >= '0' && str[0] <= '9') {
		// 0x1234 or 1234 or 123 etc
		num_bytes, val := parseIntData(str)
		if num_bytes == 2 {
			bytes = append(bytes, uint8(val>>8))
		}
		bytes = append(bytes, uint8(val&0xff))

	} else if str[0] == '"' && str[len(str)-1] == '"' {
		// "this is a string\n"
		var escapes = map[uint8]uint8{
			'n': '\n',
			'r': '\r',
			't': '\t',
			'b': '\b',
			'0': 0,
		}
		str = strings.TrimLeft(strings.Trim(str, "\""), "\"")
		for i := 0; i < len(str); i++ {
			var c = str[i]
			if c == '\\' {
				if i == len(str)-1 {
					break
				}
				escape_char, escape_exists := escapes[str[i+1]]
				if escape_exists {
					bytes = append(bytes, escape_char)
					i++
				}
				continue
			}
			bytes = append(bytes, c)
		}
	}
	return bytes
}

func parseStatements(statements []*statement, labels map[string]uint16, print_debug bool) (map[uint16][]byte, error) {
	var address uint16 = 0
	data := map[uint16][]byte{}

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
			if idx == 0 {
				// instruction or directive
				if strings.Index(arg_str, "0x") == 0 {
					// data declaration (0x1234 "this is a test\n")
					data_offset, _ := strconv.ParseUint(arg_str[2:], 16, 0)
                    data_string := strings.Join(s.source[1:], " ")
					data_bytes := parseData(data_string)
					data[uint16(data_offset)] = data_bytes
					break
				} else if arg_str[0] == '~' && len(s.source) == 2 {
					// alias (~name 0x0000)
					break
                } else if arg_str[len(arg_str) - 1] == ':' {
                    break
                } else {
                }

			} else {
				// argument
				if isArgRegister(arg_str) {
					// register (store 1 byte)
					arg_int, _ = strconv.ParseUint(arg_str[1:], 10, 0)
					s.byte_code[s.byte_count] = uint8(arg_int)

				} else if isArgImmediate(arg_str) {
					// immediate value (store 2 bytes and change to imm instruction)
                    _, imm_value := parseIntData(arg_str)
					instruction = name2instr[fmt.Sprintf("%si", instruction.name)]
					s.byte_code[0] = instruction.opcode
					s.byte_code[s.byte_count] = uint8(imm_value >> 8)
					s.byte_count++
					s.byte_code[s.byte_count] = uint8(imm_value & 0x00ff)

				} else if isArgLabel(arg_str) {
					// labels
                    label_name := arg_str[1:]
                    if label_name[0] == '.' {
                        label_name = fmt.Sprintf("%s%s", s.label, label_name)
                    }
					arg_int, _ = strconv.ParseUint(fmt.Sprintf("%04x", labels[label_name]), 16, 0)
					instruction = name2instr[fmt.Sprintf("%si", instruction.name)]
					s.byte_code[0] = instruction.opcode
					s.byte_code[s.byte_count] = uint8(arg_int >> 8)
					s.byte_count++
					s.byte_code[s.byte_count] = uint8(arg_int & 0x00ff)
				}
			}
			s.byte_count++
		}

        s.address = address
        address += 4

		if print_debug {
			// debug print the original source side by side with the byte-code
			fmt.Printf("0x%04x ", address)
			sline := ""
			for _, sword := range s.source {
				sline = fmt.Sprintf("%s%s ", sline, sword)
			}
			fmt.Printf("%-20s ", sline)
			for i := 0; i < 4; i++ {
				fmt.Printf("%02x ", s.byte_code[i])
			}
			fmt.Printf("\n")
		}
	}
	return data, nil
}

func isArgRegister(arg_str string) bool {
    return arg_str[0] == 'r' || arg_str[0] == 'R'
}

func isArgImmediate(arg_str string) bool {
    return strings.ContainsAny(arg_str[0:1], "1234567890")
}

func isArgLabel(arg_str string) bool {
    return arg_str[0] == '@'
}

func readSourceFile(filename string) ([]string, error) {
	source := make([]string, 1)
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return source, errors.New("Source file not readable\n")
	}
	source_string := string(bytes[:])
	source_lines := strings.Split(source_string, "\n")
	for line_num, line := range source_lines {
		if strings.Index(line, "%include") != 0 {
			source = append(source, line)
			continue
		}
		line_words := strings.Split(line, " ")
		include_filename := strings.Trim(strings.TrimLeft(line_words[1][1:len(line_words[1])-1], "\""), "\"")
		include_lines, err := readSourceFile(include_filename)
		if err != nil {
			fmt.Printf("%s:%d: Include file cannot be read.\n", filename, line_num)
			return source, errors.New("Source file not readable\n")
		}
		source = append(source, include_lines...)
	}
	return source, nil
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

	// load all source files (and includes)
	source_lines, err := readSourceFile(source_file)
	if err != nil {
		return
	}

	// this will hold the working copy of the source
	statements := make([]*statement, 0)

	// for each line of source, clean the line and create statement struct
	for line_num, line := range source_lines {
		var clean = cleanString(line, commentChar)
		if len(clean) == 0 {
			continue
		}
		line_statement := new(statement)
		line_statement.line_num = uint(line_num + 1)
		line_statement.source = strings.Split(strings.Clone(clean), " ")
		statements = append(statements, line_statement)
	}

	// make maps so we can lookup any->instruction
	for _, ins := range instructions {
		opcode2instr[ins.opcode] = ins
		name2instr[ins.name] = ins
	}

	err = parseAliases(statements)
	if err != nil {
		return
	}

	// go through the program and populate the label map with addresses
	labels, _ := createLabels(statements)

	// write statement bytecode chunks
	data, err := parseStatements(statements, labels, true)
	if err != nil {
		return
	}

	bytes_out := make([]byte, 0)
	// data header
	bytes_out = append(bytes_out, 1)
	bytes_out = append(bytes_out, 1)

	// for each data, write the offset, length, and value to bytecode
	data_size := 0
	for offset, value := range data {
		length := uint16(len(value))
		bytes_out = append(bytes_out, uint8(offset>>8))
		bytes_out = append(bytes_out, uint8(offset&0xff))
		bytes_out = append(bytes_out, uint8(length>>8))
		bytes_out = append(bytes_out, uint8(length&0xff))
		data_size += 4
		for _, b := range value {
			bytes_out = append(bytes_out, b)
			data_size++
		}
	}
	bytes_out[0] = uint8(data_size >> 8)
	bytes_out[1] = uint8(data_size & 0xff)

	// concat all statement bytecode chunks into output bytecode
	for _, s := range statements {
		for i := 0; i < 4; i++ {
			bytes_out = append(bytes_out, s.byte_code[i])
		}
	}

	err = os.WriteFile(output_file, bytes_out, 0666)
	if err != nil {
		fmt.Printf("Failed to write output bytecode!\n")
	}
}
