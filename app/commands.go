package main

import "strings"

func handleCommand(value Value) []byte {
	if value.Type != '*' || len(value.Array) == 0 {
		return encodeError("ERR invalid command format")
	}
	cmd := strings.ToUpper(value.Array[0].Str)
	switch cmd {
	case "PING":
		return encodeSimpleString("PONG")
	case "ECHO":
		if len(value.Array) < 2 {
			return encodeError("ERR wrong number of arguments for 'echo' command")
		}
		return encodeBulkString(value.Array[1].Str)
	default:
		return encodeError("ERR unknown command")
	}
}
