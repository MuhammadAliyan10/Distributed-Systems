// redis/internal/server/handler.go
package server

import (
	"fmt"
	"io"
	"net"
	"redis/internal/resp"
)


func (s *Server) handleConnection(conn net.Conn){
	defer conn.Close()

	parser := resp.NewParser(conn)
	writer := resp.NewWriter(conn)


	for {

		value, err := parser.Read()
		if err != nil {
			if err == io.EOF{
				break
			}
			fmt.Println("Error reading from client:", err)
			break
		}
		if value.Type != "array" || len(value.Array) == 0 {
			writer.WriteError("ERR expected array of commands")
			continue
		}

		cmdName := value.Array[0].Bulk

		args := value.Array[1:]

		result := s.registry.Execute(cmdName, args, s.db)

		err = s.writeResult(writer, result)

if err != nil {
			fmt.Println("Error writing to client:", err)
			break
		}
	}



}



func (s *Server) writeResult(w *resp.Writer, result resp.Value) error {
	switch result.Type {
	case "string":
		return w.WriteSimpleString(result.Str)
	case "error":
		return w.WriteError(result.Str)
	case "integer":
		return w.WriteInteger(result.Num)
	case "bulk":
		return w.WriteBulkString(result.Bulk)
	case "null":
		return w.WriteNull()
	default:
		return w.WriteError("ERR unknown result type")
	}
}
