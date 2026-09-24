// redis/cmd/redis-server/main.go
package main

import (
	"fmt"
	"io"
	"os"
	"redis/internal/commands"
	"redis/internal/resp"
	"redis/internal/server"
	"redis/internal/storage/aof"
	"redis/internal/storage/memory"
	"strings"
)





func main(){

	db := memory.NewStore()
	registry := commands.NewRegistry()
	commands.RegisterStringsCommand(registry)
	commands.RegisterKeyCommands(registry)


	f, err := os.Open("database.aof")

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

	aofLog, err := aof.NewAOF("database.aof")
	if err !=nil{
		fmt.Println("Failed to initialize AOF:", err)
		os.Exit(1)
	}
defer aofLog.Close()

svr := server.NewServer(":6379", db, aofLog, registry)
err = svr.Start()


	if err != nil {
		fmt.Println("Server failed to start:", err)
		os.Exit(1)
	}
}
