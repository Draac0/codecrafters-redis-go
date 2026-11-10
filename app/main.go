package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
)

func handleConn(conn net.Conn) {
	defer conn.Close() // runs to clean up the connection after the function completion
	reader := bufio.NewReader(conn)
	for {
		value, err := readValue(reader)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				fmt.Println("handleConn read: ", err.Error())
			}
			break
		}
		writeValue := handleCommand(value)
		_, err = conn.Write(writeValue)
		if err != nil {
			fmt.Println("handleConn write: ", err.Error())
			break
		}
	}
}

func main() {
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			continue
		}
		go handleConn(conn)
	}

}
