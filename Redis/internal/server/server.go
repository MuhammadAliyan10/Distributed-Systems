// redis/internal/server/server.go
package server

import (
	"fmt"
	"net"
	"redis/internal/commands"
	"redis/internal/replication"
	"redis/internal/resp"
	"redis/internal/storage"
	"redis/internal/storage/aof"
)

type Server struct{
	addr string
	db storage.Engine
	registry *commands.Registry
	aofLog *aof.AOF
	broker *replication.Broker
}


func NewServer(addr string, db storage.Engine, aofLog *aof.AOF, registry *commands.Registry, broker *replication.Broker) *Server{
	return &Server{
		addr: addr,
		db: db,
		registry: registry,
		aofLog: aofLog,
		broker: broker,
	}
}

func (s *Server) Start() error{
	listner, err := net.Listen("tcp", s.addr)

	if err != nil{
		return err
	}

	defer listner.Close()

	fmt.Println("Redis is running on", s.addr)


	for {
		conn, err := listner.Accept()
		if err != nil{
fmt.Println("Error accepting connection:", err)
continue
		}

		go s.handleConnection(conn)
	}
}


func (s *Server) StartReplicaClient(masterAddr string){
	fmt.Println("Booting in Replica Mode. Connecting to Master at", masterAddr)

	conn, err := net.Dial("tcp", masterAddr)

	if err != nil{
		fmt.Println("Replica failed to connect to master:", err)
		return
	}
conn.Write([]byte("*1\r\n$4\r\nSYNC\r\n"))
parser := resp.NewParser(conn)

parser.Read()
fmt.Println("Synced with Master! Listening for broadcasts...")

for {
	val, err := parser.Read()
	if err != nil{
		fmt.Println("Lost connection to Master.")
			break
	}

	cmdName := val.Array[0].Bulk
	args := val.Array[1:]
	s.registry.Execute(cmdName, args, s.db)
}

}
