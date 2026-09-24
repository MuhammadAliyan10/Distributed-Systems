// redis/internal/server/server.go
package server

import (
	"fmt"
	"net"
	"redis/internal/commands"
	"redis/internal/storage"
	"redis/internal/storage/aof"
)

type Server struct{
	addr string
	db storage.Engine
	registry *commands.Registry
	aofLog *aof.AOF
}


func NewServer(addr string, db storage.Engine, aofLog *aof.AOF, registry *commands.Registry) *Server{
	return &Server{
		addr: addr,
		db: db,
		registry: registry,
		aofLog: aofLog,
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
