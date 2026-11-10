package main

import "testing"

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
	tests := []struct {
		name   string
		input  Value
		want   string
		preset map[string]string
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
			name:   "GET cmd",
			input:  cmd("GET", "name"),
			want:   "$5\r\nAlice\r\n",
			preset: map[string]string{"name": "Alice"},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStore()
			for k, v := range tt.preset {
				store.Set(k, v)
			}

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
