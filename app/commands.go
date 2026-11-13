package main

import (
	"strconv"
	"strings"
	"time"
)

func echoCmd(value Value) []byte {
	if len(value.Array) < 2 {
		return encodeError("ERR wrong number of arguments for 'echo' command")
	}
	return encodeBulkString(value.Array[1].Str)
}

func getDuration(expType string, duration int) time.Duration {
	if expType == "EX" {
		return time.Duration(duration) * time.Second
	}
	return time.Duration(duration) * time.Millisecond
}

func setCmd(value Value, store *Store) []byte {
	args := value.Array
	var duration time.Duration

	switch len(args) {
	case 3:
		// key, value pair
	case 5:
		exp := strings.ToUpper(args[3].Str)
		if exp != "EX" && exp != "PX" {
			return encodeError("ERR syntax error")
		}
		dur, err := strconv.Atoi(args[4].Str)
		if err != nil {
			return encodeError("ERR value is not an integer or out of range")
		}
		if dur < 1 {
			return encodeError("ERR invalid expire time in 'set' command")
		}
		duration = getDuration(exp, dur)
	default:
		return encodeError("ERR wrong number of arguments for 'set' command")
	}
	store.Set(args[1].Str, args[2].Str, duration)
	return encodeSimpleString("OK")
}

func getCmd(value Value, store *Store) []byte {
	if len(value.Array) != 2 {
		return encodeError("ERR wrong number of arguments for 'get' command")
	}
	key := value.Array[1].Str
	data, ok := store.Get(key)
	if !ok {
		return encodeNullBulkString()
	}
	return encodeBulkString(data)
}

func rpushCmd(value Value, store *Store) []byte {
	if len(value.Array) < 3 {
		return encodeError("ERR wrong number of arguments for 'rpush' command")
	}
	key := value.Array[1].Str
	vals := make([]string, len(value.Array)-2)
	for i, v := range value.Array[2:] {
		vals[i] = v.Str
	}
	l, err := store.RPush(key, vals...)
	if err != nil {
		return encodeError(err.Error())
	}	
	return encodeInteger(int64(l))
}

func handleCommand(value Value, store *Store) []byte {
	if value.Type != '*' || len(value.Array) == 0 {
		return encodeError("ERR invalid command format")
	}
	cmd := strings.ToUpper(value.Array[0].Str)
	switch cmd {
	case "PING":
		return encodeSimpleString("PONG")
	case "ECHO":
		return echoCmd(value)
	case "SET":
		return setCmd(value, store)
	case "GET":
		return getCmd(value, store)
	case "RPUSH":
		return rpushCmd(value, store)
	default:
		return encodeError("ERR unknown command")
	}
}
