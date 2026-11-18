package main

import (
	"errors"
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
	data, err := store.Get(key)
	if err != nil {
		if errors.Is(err, ErrWrongType) {
			return encodeError(err.Error())
		}
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

func lrangeCmd(value Value, store *Store) []byte {
	if len(value.Array) != 4 {
		return encodeError("ERR wrong number of arguments for 'lrange' command")
	}
	arr := value.Array
	key := arr[1].Str
	start, err := strconv.Atoi(arr[2].Str)
	if err != nil {
		return encodeError("ERR value is not an integer or out of range")
	}
	end, err := strconv.Atoi(arr[3].Str)
	if err != nil {
		return encodeError("ERR value is not an integer or out of range")
	}
	list, err := store.LRange(key, start, end)
	if err != nil {
		return encodeError(err.Error())
	}
	return encodeStringArray(list)
}

func lpushCmd(value Value, store *Store) []byte {
	if len(value.Array) < 3 {
		return encodeError("ERR wrong number of arguments for 'lpush' command")
	}
	key := value.Array[1].Str
	vals := make([]string, len(value.Array)-2)
	for i, v := range value.Array[2:] {
		vals[i] = v.Str
	}
	l, err := store.LPush(key, vals...)
	if err != nil {
		return encodeError(err.Error())
	}
	return encodeInteger(int64(l))
}

func llenCmd(value Value, store *Store) []byte {
	if len(value.Array) != 2 {
		return encodeError("ERR wrong number of arguments for 'llen' command")
	}
	key := value.Array[1].Str
	l, err := store.LLen(key)
	if err != nil {
		return encodeError(err.Error())
	}
	return encodeInteger(int64(l))
}

func lpopCmd(value Value, store *Store) []byte {
	if len(value.Array) < 2 || len(value.Array) > 3 {
		return encodeError("ERR wrong number of arguments for 'lpop' command")
	}
	key := value.Array[1].Str
	count := 1
	hasCountArg := false
	if len(value.Array) == 3 {
		c, err := strconv.Atoi(value.Array[2].Str)
		if err != nil || c < 0 {
			return encodeError("ERR value is out of range, must be positive")
		}
		count = c
		hasCountArg = true
	}
	elements, ok, err := store.LPop(key, count)
	if err != nil {
		return encodeError(err.Error())
	}
	if !ok {
		if hasCountArg {
			return encodeNullArray()
		}
		return encodeNullBulkString()
	}
	if hasCountArg {
		return encodeStringArray(elements)
	}
	return encodeBulkString(elements[0])
}

func blpopCmd(value Value, store *Store) []byte {
	if len(value.Array) < 3 {
		return encodeError("ERR wrong number of arguments for 'blpop' command")
	}
	key := value.Array[1].Str
	seconds, err := strconv.ParseFloat(value.Array[2].Str, 64)
	if err != nil {
		return encodeError("ERR timeout is not a float or out of range")
	}
	if seconds < 0 {
		return encodeError("ERR timeout is negative")
	}
	elements, hasTimedOut, err := store.BLPop(key, int(seconds * 1000))
	if err != nil {
		return encodeError(err.Error())
	}
	if hasTimedOut {
		return encodeNullArray()
	}
	return encodeStringArray(elements)
}

func typeCmd(value Value, store *Store) []byte {
	if len(value.Array) != 2 {
		return encodeError("ERR wrong number of arguments for 'type' command")
	}
	key := value.Array[1].Str
	t := store.Type(key)
	return encodeSimpleString(t)
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
	case "LPUSH":
		return lpushCmd(value, store)
	case "LRANGE":
		return lrangeCmd(value, store)
	case "LLEN":
		return llenCmd(value, store)
	case "LPOP":
		return lpopCmd(value, store)
	case "BLPOP":
		return blpopCmd(value, store)
	case "TYPE":
		return typeCmd(value, store)
	default:
		return encodeError("ERR unknown command")
	}
}
