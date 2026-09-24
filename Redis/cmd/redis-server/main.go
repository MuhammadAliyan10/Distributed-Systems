// redis/cmd/redis-server/main.go
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"redis/internal/commands"
	"redis/internal/replication"
	"redis/internal/resp"
	"redis/internal/server"
	"redis/internal/storage/aof"
	"redis/internal/storage/memory"
	"strings"
)





func main(){

	port := flag.String("port", "6379", "Port to run the server on")
replicaOf := flag.String("replicaof", "", "Master address to replicate from")
flag.Parse()

	aofFileName := "database_"+ *port + ".aof"
	db := memory.NewStore()
	registry := commands.NewRegistry()
	commands.RegisterStringsCommand(registry)
	commands.RegisterKeyCommands(registry)




	f, err := os.Open(aofFileName)

	if err == nil {
		fmt.Println("Restoring database from AOF log...")
		parser := resp.NewParser(f)

		for {
			val, err := parser.Read()
			if err != nil{
				if err == io.EOF{
					break
				}
				fmt.Println("Error reading AOF:", err)
				break
			}
			if val.Type == "array" && len(val.Array) > 0 {
				cmdName := strings.ToUpper(val.Array[0].Bulk)
				args := val.Array[1:]
				registry.Execute(cmdName, args, db)
			}
		}
		f.Close()
		fmt.Println("Database restored successfully.")
	}

	aofLog, err := aof.NewAOF(aofFileName)
	if err !=nil{
		panic(err)

	}
defer aofLog.Close()

broker := replication.NewBroker()

svr := server.NewServer(":"+*port, db, aofLog, registry,broker)

if *replicaOf != ""{
	go svr.StartReplicaClient(*replicaOf)
}
fmt.Println("Redis is running on :" + *port)
err = svr.Start()


	if err != nil {
		fmt.Println("Server failed to start:", err)
		os.Exit(1)
	}
}
