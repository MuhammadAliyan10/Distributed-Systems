// redis/cmd/redis-server/main.go
package main

import (
	"fmt"
	"os"
	"redis/internal/commands"
	"redis/internal/server"
	"redis/internal/storage/memory"
)





func main(){

	db := memory.NewStore()

	registry := commands.NewRegistry()

	commands.RegisterStringsCommand(registry)

	srv := server.NewServer(":6379", db, registry)

	err:= srv.Start()

	if err != nil {
		fmt.Println("Server failed to start:", err)
		os.Exit(1)
	}
}
