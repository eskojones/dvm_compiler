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
	{"nop", 0x00, 1},
	{"hlt", 0xff, 1},
	{"print", 0xfe, 2},
	{"ld", 0x01, 4},
	{"st", 0x02, 4},
	{"int", 0x03, 2},
	{"inti", 0x04, 3},
	{"ret", 0x05, 1},
	{"mov", 0x06, 3},
	{"movi", 0x07, 4},
	{"cmp", 0x08, 3},
	{"cmpi", 0x09, 4},
	{"jmp", 0x0a, 2},
	{"jmpi", 0x0b, 3},
	{"jz", 0x0c, 2},
	{"jzi", 0x0d, 3},
	{"jnz", 0x0e, 2},
	{"jnzi", 0x0f, 3},
	{"jl", 0x10, 2},
	{"jli", 0x11, 3},
	{"jg", 0x12, 2},
	{"jgi", 0x13, 3},
	{"inc", 0x14, 2},
	{"dec", 0x15, 2},
	{"add", 0x16, 3},
	{"addi", 0x17, 4},
	{"sub", 0x18, 3},
	{"subi", 0x19, 4},
	{"mul", 0x1a, 3},
	{"muli", 0x1b, 4},
	{"div", 0x1c, 3},
	{"divi", 0x1d, 4},
	{"call", 0x1e, 2},
	{"calli", 0x1f, 3},
}

type statement struct {
	line_num   uint
	address    uint16
	source     []string
	byte_code  [6]byte
	byte_count int
}

var opcode2instr = map[uint8]instr{}
var name2instr = map[string]instr{}

func cleanString(dirty string, comment string) string {
	var clean string = strings.Clone(dirty)
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
					return errors.New("Invalid alias referenced")
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

		if instr_name[0] == '.' || instr_name[0] == '~' {
			//data or alias (ignore this until parse)
			continue
		}

		_, exists := name2instr[instr_name]
		if !exists {
			fmt.Printf("Line %d: Invalid instruction (%s)\n", s.line_num, instr_name)
			return labels, errors.New("Invalid Instruction")
		}

		s.byte_count = 0
		isImmediate := false

		if len(s.source) > 3 {
			fmt.Printf("Line %d: Too many arguments for statement \"%s\".\n", s.line_num, instr_name)
			return labels, errors.New("Too many arguments in statement")
		}

		for idx, arg_str := range s.source {
			if idx > 0 {
				if strings.ContainsAny(arg_str[0:1], "0123456789") {
					if isImmediate {
						fmt.Printf("Line %d: Multiple immediate values, this is invalid.\n", s.line_num)
						return labels, errors.New("Multiple Immediate Values")
					}
					s.byte_count++
					isImmediate = true
				} else if arg_str[0] == '@' {
					if isImmediate {
						fmt.Printf("Line %d: Multiple immediate values, this is invalid.\n", s.line_num)
						return labels, errors.New("Multiple Immediate Values")
					}
					s.byte_count++
					isImmediate = true
				}
			}

			s.byte_count++
		}
		address += uint16(s.byte_count)
	}

	return labels, nil
}

func parseIntData(data string) (uint8, uint16) {
	var intval uint64
	if strings.Index(data, "0x") == 0 {
		intval, _ = strconv.ParseUint(data[2:], 16, 0)
	} else {
		intval, _ = strconv.ParseUint(data, 10, 0)
	}
	byte_count := uint8(1)
	if intval > 0xff {
		byte_count = 2
	}
	return byte_count, uint16(intval)
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
			var c uint8 = str[i]
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
				if strings.Index(arg_str, ".0x") == 0 {
					// data declaration (.0x1234 "this is a test\n")
					data_offset, _ := strconv.ParseUint(arg_str[3:], 16, 0)
					data_string := strings.Join(s.source[1:], " ")
					data_bytes := parseData(data_string)
					data[uint16(data_offset)] = data_bytes
					break
				} else if arg_str[0] == '~' && len(s.source) == 2 {
					// alias (~name 0x0000)
					break
				}

			} else {
				// argument
				if arg_str[0] == 'r' || arg_str[0] == 'R' {
					// register
					arg_int, _ = strconv.ParseUint(arg_str[1:], 10, 0)
					s.byte_code[s.byte_count] = uint8(arg_int)

				} else if strings.ContainsAny(arg_str[0:1], "0123456789") {
					// immediate value
					if strings.Index(arg_str, "0x") == 0 {
						arg_int, _ = strconv.ParseUint(arg_str[2:], 16, 0)
					} else if arg_str[len(arg_str)-1] == 'h' {
						arg_int, _ = strconv.ParseUint(arg_str[:len(arg_str)-1], 16, 0)
					} else {
						arg_int, _ = strconv.ParseUint(arg_str, 10, 0)
					}
					instruction = name2instr[fmt.Sprintf("%si", instruction.name)]
					s.byte_code[0] = instruction.opcode
					s.byte_code[s.byte_count] = uint8(arg_int >> 8)
					s.byte_count++
					s.byte_code[s.byte_count] = uint8(arg_int & 0x00ff)

				} else if arg_str[0] == '@' {
					// label
					arg_int, _ = strconv.ParseUint(fmt.Sprintf("%04x", labels[arg_str[1:]]), 16, 0)
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
	return data, nil
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
		var clean string = cleanString(line, "#")
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
		for i := 0; i < s.byte_count; i++ {
			bytes_out = append(bytes_out, s.byte_code[i])
		}
	}

	err = os.WriteFile(output_file, bytes_out, 0666)
	if err != nil {
		fmt.Printf("Failed to write output bytecode!\n")
	}
}
