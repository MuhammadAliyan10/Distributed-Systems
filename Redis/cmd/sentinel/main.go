package main

import (
	"fmt"
	"net"
	"time"
)



func main(){
fmt.Println("Sentinel Node starting up...")
masterAddress := "127.0.0.1:6379"

for {
	time.Sleep(2 * time.Second)

	conn, err := net.DialTimeout("tcp", masterAddress, 1*time.Second)

	if err != nil{
	fmt.Println("CRITICAL: Master is DOWN! Connection refused.")
		fmt.Println("Initiating Failover... Contacting Replica...")

		replicaConn, replErr := net.Dial("tcp", "127.0.0.1:6380")
		if replErr == nil{
		replicaConn.Write([]byte("*1\r\n$7\r\nPROMOTE\r\n"))
			fmt.Println("Failover Complete! Replica has been promoted to Master.")
			replicaConn.Close()

			masterAddress = "127.0.0.1:6380"
			continue
		}else{
			fmt.Println("FATAL: Replica is also down. Cluster is dead.")
		}



	}

	_, err = conn.Write([]byte("*1\r\n$4\r\nPING\r\n"))
	if err != nil{
		fmt.Println("CRITICAL: Master is DOWN! Write failed.")
			conn.Close()
			continue
	}

	buffer := make([]byte, 1024)
	_, err = conn.Read(buffer)

	if err != nil{
		fmt.Println("CRITICAL: Master is DOWN! Read failed.")
			conn.Close()
			continue
	}
fmt.Println("Heartbeat OK: Master is healthy.")
		conn.Close() // Close the connection until the next heartbeat

}
}
