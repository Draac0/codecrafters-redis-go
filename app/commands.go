package main

import (
	"strings"
)

func handleCommand(value Value, store *Store) []byte {
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
	case "SET":
		if len(value.Array) != 3 {
			return encodeError("ERR wrong number of arguments for 'set' command")
		}
		key := value.Array[1].Str
		val := value.Array[2].Str
		store.Set(key, val)
		return encodeSimpleString("OK")
	case "GET":
		if len(value.Array) != 2 {
			return encodeError("ERR wrong number of arguments for 'get' command")
		}
		key := value.Array[1].Str
		data, ok := store.Get(key)
		if !ok {
			return encodeNullBulkString()
		}
		return encodeBulkString(data)
	default:
		return encodeError("ERR unknown command")
	}
}
