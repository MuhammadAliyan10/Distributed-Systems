// redis/internal/server/handler.go
package server

import (
	"fmt"
	"io"
	"net"
	"redis/internal/resp"
	"strconv"
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

		if cmdName == "SYNC" {
			s.broker.AddReplica(conn)
			writer.WriteSimpleString("OK SYNCING")

			continue
		}

		if cmdName == "BGREWRITEAOF"{
			err := s.aofLog.Rewrite(s.db)
			if err != nil{
				writer.WriteError("ERR " + err.Error())
			} else {
				writer.WriteSimpleString("OK REWRITE COMPLETE")
			}
			continue
		}

		if cmdName == "PROMOTE"{
			writer.WriteSimpleString("OK PROMOTED TO MASTER")

			continue
		}

		args := value.Array[1:]

		result := s.registry.Execute(cmdName, args, s.db)

		if result.Type != "error" && (cmdName == "SET" || cmdName == "DEL"){
				rawBytes := marshalRESPArray(value.Array)
				s.aofLog.Write(rawBytes)

		   	s.broker.Broadcast(rawBytes)
		}

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


func marshalRESPArray(array []resp.Value) []byte{
	var result []byte
	result = append(result, '*')
	result = append(result, strconv.Itoa(len(array))...)
	result = append(result,'\r', '\n' )

	for _, val := range array {
		result = append(result, '$')
		result = append(result, strconv.Itoa(len(val.Bulk))...)
			result = append(result, '\r', '\n')
		result = append(result, val.Bulk...)
		result = append(result, '\r', '\n')

	}
		return result
}
