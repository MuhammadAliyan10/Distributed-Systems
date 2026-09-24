package main

import (
	"fmt"
	"io"
	"net"
	"os"
)

func main() {
	fmt.Println("Replica Node starting up...")

	go func(){
		conn, err := net.Dial("tcp", "127.0.0.1:6379")
		if err == nil{
			conn.Write([]byte("*1\r\n$4\r\nSYNC\r\n"))
	fmt.Println("Connected to Master! Listening for broadcasts...")
			io.Copy(os.Stdout, conn)
		}
	}()

	listener, err := net.Listen("tcp", ":6380")
	if err != nil {
		fmt.Println("Failed to start replica listener:", err)
		os.Exit(1)
	}
		fmt.Println("Replica management port open on :6380")


	for {
		conn, err := listener.Accept()
		if err != nil{
			continue
		}
		buffer := make([]byte, 1024)
		n, _ := conn.Read(buffer)
		command := string(buffer[:n])

		if command == "PROMOTE" {
			fmt.Println("\n RECEIVED PROMOTE COMMAND FROM SENTINEL")
			fmt.Println("I am taking control. I am the new Master now!")
			// In a full implementation, the Replica would boot up its TCP server on :6379 here
		}
		conn.Close()
	}
}
