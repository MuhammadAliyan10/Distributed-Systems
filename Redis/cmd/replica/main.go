package main

import (
	"fmt"
	"io"
	"net"
	"os"
)

func main() {
	fmt.Println("Replica Node starting up...")
	fmt.Println("Connecting to Master on :6379...")
	
	// Dial the Master server over TCP
	conn, err := net.Dial("tcp", "127.0.0.1:6379")
	if err != nil {
		fmt.Println("Failed to connect to master:", err)
		os.Exit(1)
	}

	// Send the SYNC command using proper raw RESP format
	_, err = conn.Write([]byte("*1\r\n$4\r\nSYNC\r\n"))
	if err != nil {
		fmt.Println("Failed to send SYNC command:", err)
		os.Exit(1)
	}

	fmt.Println("Connected! Listening for Master broadcasts (Streaming Mode)...")

	// This instantly prints everything the Master streams to us to the console
	// In a full implementation, we would pass this stream to our `parser.Read()`!
	io.Copy(os.Stdout, conn)
}
