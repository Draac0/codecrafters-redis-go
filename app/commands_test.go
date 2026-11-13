package main

import (
	"maps"
	"testing"
	"time"
)

func cmd(args ...string) Value {
	val := Value{
		Type: '*',
	}
	val.Array = make([]Value, len(args))
	for i, v := range args {
		val.Array[i] = Value{
			Type: '$',
			Str:  v,
		}
	}
	return val
}

func TestHandleCommand(t *testing.T) {
	abc := map[string]entry{
		"list": {kind: kindList, list: []string{"a", "b", "c"}},
	}
	tests := []struct {
		name   string
		input  Value
		want   string
		preset map[string]entry
	}{
		{
			name:  "invalid value type",
			input: Value{Type: '+'},
			want:  "-ERR invalid command format\r\n",
		},
		{
			name:  "array len 0",
			input: cmd(),
			want:  "-ERR invalid command format\r\n",
		},
		{
			name:  "cmd PING",
			input: cmd("PING"),
			want:  "+PONG\r\n",
		},
		{
			name:  "cmd check case-sensitive ping",
			input: cmd("ping"),
			want:  "+PONG\r\n",
		},
		{
			name:  "cmd check ECHO",
			input: cmd("ECHO", "hey"),
			want:  "$3\r\nhey\r\n",
		},
		{
			name:  "cmd check no-args ECHO",
			input: cmd("ECHO"),
			want:  "-ERR wrong number of arguments for 'echo' command\r\n",
		},
		{
			name:  "unknown cmd",
			input: cmd("FOO"),
			want:  "-ERR unknown command\r\n",
		},
		{
			name:  "SET cmd",
			input: cmd("SET", "name", "Alice"),
			want:  "+OK\r\n",
		},
		{
			name:  "SET cmd: out of range values in array",
			input: cmd("SET", "name", "Alice", "PX"),
			want:  "-ERR wrong number of arguments for 'set' command\r\n",
		},
		{
			name:  "SET cmd: expires at 1 ms",
			input: cmd("SET", "name", "Alice", "PX", "1"),
			want:  "+OK\r\n",
		},
		{
			name:  "GET cmd",
			input: cmd("GET", "name"),
			want:  "$5\r\nAlice\r\n",
			preset: map[string]entry{
				"name": {
					str:  "Alice",
					kind: kindString,
				},
			},
		},
		{
			name:  "expired key: GET cmd",
			input: cmd("GET", "name"),
			want:  "$-1\r\n",
			preset: map[string]entry{
				"name": {
					str:       "Alice",
					expiresAt: time.Now().Add(-time.Second),
				},
			},
		},
		{
			name:  "invalid arguments for SET",
			input: cmd("SET", "name"),
			want:  "-ERR wrong number of arguments for 'set' command\r\n",
		},
		{
			name:  "invalid arguments for GET",
			input: cmd("GET"),
			want:  "-ERR wrong number of arguments for 'get' command\r\n",
		},
		{
			name:  "invalid arguments for RPUSH",
			input: cmd("RPUSH"),
			want:  "-ERR wrong number of arguments for 'rpush' command\r\n",
		},
		{
			name:  "set string and change it to RPUSH",
			input: cmd("RPUSH", "key", "value"),
			want:  "-WRONGTYPE Operation against a key holding the wrong kind of value\r\n",
			preset: map[string]entry{
				"key": {
					str:  "123",
					kind: 's',
				},
			},
		},
		{
			name:  "cmd: rpush, set one value",
			input: cmd("RPUSH", "key", "value"),
			want:  ":1\r\n",
		},
		{
			name:  "cmd: rpush, set three value",
			input: cmd("RPUSH", "key", "value", "value 1", "value 2"),
			want:  ":3\r\n",
		},
		{
			name:  "cmd: rpush, append a value",
			input: cmd("RPUSH", "key", "value"),
			want:  ":3\r\n",
			preset: map[string]entry{
				"key": {
					kind: 'l',
					list: []string{"one", "two"},
				},
			},
		},
		{
			name:  "invalid arguments for LRANGE",
			input: cmd("LRANGE"),
			want:  "-ERR wrong number of arguments for 'lrange' command\r\n",
		},
		{
			name:  "invalid arguments for LRANGE (non integer values)",
			input: cmd("LRANGE", "key", "ab", "1"),
			want:  "-ERR value is not an integer or out of range\r\n",
		},
		{
			name:   "LRANGE first two",
			input:  cmd("LRANGE", "list", "0", "1"),
			want:   "*2\r\n$1\r\na\r\n$1\r\nb\r\n",
			preset: abc,
		},
		{
			name:   "LRANGE single element",
			input:  cmd("LRANGE", "list", "1", "1"),
			want:   "*1\r\n$1\r\nb\r\n",
			preset: abc,
		},
		{
			name:   "LRANGE whole list with -1",
			input:  cmd("LRANGE", "list", "0", "-1"),
			want:   "*3\r\n$1\r\na\r\n$1\r\nb\r\n$1\r\nc\r\n",
			preset: abc,
		},
		{
			name:   "LRANGE last two",
			input:  cmd("LRANGE", "list", "-2", "-1"),
			want:   "*2\r\n$1\r\nb\r\n$1\r\nc\r\n",
			preset: abc,
		},
		{
			name:   "LRANGE last only",
			input:  cmd("LRANGE", "list", "-1", "-1"),
			want:   "*1\r\n$1\r\nc\r\n",
			preset: abc,
		},
		{
			name:   "LRANGE start far below zero is clamped",
			input:  cmd("LRANGE", "list", "-10", "1"),
			want:   "*2\r\n$1\r\na\r\n$1\r\nb\r\n",
			preset: abc,
		},
		{
			name:   "LRANGE stop far below zero",
			input:  cmd("LRANGE", "list", "0", "-10"),
			want:   "*0\r\n",
			preset: abc,
		},

		// --- out of range ---
		{
			name:   "LRANGE stop past end is clamped",
			input:  cmd("LRANGE", "list", "0", "10"),
			want:   "*3\r\n$1\r\na\r\n$1\r\nb\r\n$1\r\nc\r\n",
			preset: abc,
		},
		{
			name:   "LRANGE start past end",
			input:  cmd("LRANGE", "list", "5", "10"),
			want:   "*0\r\n",
			preset: abc,
		},
		{
			name:   "LRANGE start after stop",
			input:  cmd("LRANGE", "list", "2", "1"),
			want:   "*0\r\n",
			preset: abc,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStore()
			maps.Copy(store.data, tt.preset)

			got := handleCommand(tt.input, store)
			if string(got) != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
				return
			}
		})
	}
}

func TestSetOverwrite(t *testing.T) {
	store := NewStore()

	handleCommand(cmd("SET", "name", "Alice"), store)
	handleCommand(cmd("SET", "name", "Bob"), store)

	got := handleCommand(cmd("GET", "name"), store)
	want := "$3\r\nBob\r\n"

	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
