package main

import (
	"bufio"
	"reflect"
	"strings"
	"testing"
)

func TestReadValue(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Value
		wantErr bool
	}{
		{
			name:  "simple string",
			input: "+OK\r\n",
			want:  Value{Type: '+', Str: "OK"},
		},
		{
			name:    "unknown type byte",
			input:   "x\r\n",
			wantErr: true,
		},
		{
			name:  "simple integer",
			input: ":42\r\n",
			want:  Value{Type: ':', Int: 42},
		},
		{
			name:  "simple negative integer",
			input: ":-7\r\n",
			want:  Value{Type: ':', Int: -7},
		},
		{
			name:  "bulk string",
			input: "$5\r\nhello\r\n",
			want:  Value{Type: '$', Str: "hello"},
		},
		{
			name:  "empty bulk string",
			input: "$0\r\n\r\n",
			want:  Value{Type: '$', Str: ""},
		},
		{
			name:  "null bulk string",
			input: "$-1\r\n",
			want:  Value{Type: '$', Null: true},
		},
		{
			name:  "bulk string containing CRLF",
			input: "$7\r\nab\r\ncde\r\n",
			want:  Value{Type: '$', Str: "ab\r\ncde"},
		},
		{
			name:  "array",
			input: "*2\r\n$4\r\nECHO\r\n$3\r\nhey\r\n",
			want: Value{Type: '*', Array: []Value{
				{
					Type: '$',
					Str:  "ECHO",
				},
				{
					Type: '$',
					Str:  "hey",
				},
			}},
		},
		{
			name:  "null array",
			input: "*-1\r\n",
			want:  Value{Type: '*', Null: true},
		},
		{
			name:  "error type",
			input: "-ERR Bad\r\n",
			want:  Value{Type: '-', Str: "ERR Bad"},
		},
		{
			name:  "empty array",
			input: "*0\r\n",
			want:  Value{Type: '*', Array: []Value{}},
		},
		{
			name:  "nested array",
			input: "*2\r\n*1\r\n$1\r\na\r\n:5\r\n",
			want: Value{Type: '*', Array: []Value{
				{
					Type: '*',
					Array: []Value{
						{
							Type: '$',
							Str:  "a",
						},
					},
				},
				{
					Type: ':',
					Int:  5,
				},
			}},
		},
		{
			name:    "bad integer",
			input:   ":abc\r\n",
			wantErr: true,
		},
		{
			name:    "bad bulk length",
			input:   "$-5\r\n",
			wantErr: true,
		},
		{
			name:    "bad array length",
			input:   "*-5\r\n",
			wantErr: true,
		},
		{
			name:    "missing \\r",
			input:   "+OK\n",
			wantErr: true,
		},
		{
			name:    "cut-off bulk string",
			input:   "$5\r\nhel",
			wantErr: true,
		},
		{
			name:    "wrong bulk ending",
			input:   "$5\r\nhelloXX",
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
		// {
		// 	name:  "",
		// 	input: "",
		// 	wantErr: true,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(strings.NewReader(tt.input))
			got, err := readValue(r)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected an error, got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error %v", err)
				return
			}
			if !reflect.DeepEqual(tt.want, got) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

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
		name  string
		input Value
		want  string
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handleCommand(tt.input)
			if string(got) != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
				return
			}
		})
	}
}
