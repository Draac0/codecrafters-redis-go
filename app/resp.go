package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Value struct {
	Type  byte // + (simple strings) - (errors) : (int) $
	Str   string
	Int   int64
	Null  bool
	Array []Value
}

func readValue(r *bufio.Reader) (Value, error) {
	t, err := r.ReadByte()
	value := Value{}
	if err != nil {
		return value, err
	}

	line, err := readLine(r)
	if err != nil {
		return value, err
	}
	value.Type = t
	switch t {
	case '+', '-':
		value.Str = line
	case ':':
		i, err := strconv.ParseInt(line, 10, 64)
		if err != nil {
			return value, err
		}
		value.Int = i
	case '$':
		length, err := strconv.Atoi(line)
		if err != nil {
			return value, err
		}
		if length == -1 {
			value.Null = true
		} else if length < -1 {
			return value, fmt.Errorf("invalid bulk string length: %d", length)
		} else {
			buf := make([]byte, length+2)
			_, err = io.ReadFull(r, buf)
			if err != nil {
				return value, err
			}
			if buf[length] != '\r' || buf[length+1] != '\n' {
				return value, errors.New("read failed: unknown end character")
			}
			value.Str = string(buf[:length])
		}
	case '*':
		length, err := strconv.Atoi(line)
		if err != nil {
			return value, err
		}
		if length == -1 {
			value.Null = true
		} else if length < -1 {
			return value, fmt.Errorf("invalid array length: %d", length)
		} else {
			arr := make([]Value, length)
			for i := range length {
				val, err := readValue(r)
				if err != nil {
					return value, fmt.Errorf("invalid array value %w", err)
				}
				arr[i] = val
			}
			value.Array = arr
		}
	default:
		return value, fmt.Errorf("unknown type byte: %q", t)
	}
	return value, nil
}

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	if !strings.HasSuffix(line, "\r\n") {
		return "", errors.New("line doesn't end with \\r\\n")
	}
	return line[:len(line)-2], nil
}

func encodeSimpleString(s string) []byte {
	return fmt.Appendf(nil, "+%s\r\n", s)
}

func encodeBulkString(s string) []byte {
	return fmt.Appendf(nil, "$%d\r\n%s\r\n", len(s), s)
}

func encodeNullBulkString() []byte {
	return []byte("$-1\r\n")
}

func encodeError(msg string) []byte {
	return fmt.Appendf(nil, "-%s\r\n", msg)
}

func encodeInteger(n int64) []byte {
	return fmt.Appendf(nil, ":%d\r\n", n)
}

func encodeStringArray(str []string) []byte {
	var s []byte = []byte(fmt.Appendf(nil, "*%d\r\n", len(str)))
	for _, st := range str {
		s = fmt.Appendf(s, "$%d\r\n%s\r\n", len(st), st)
	}
	return s
}
