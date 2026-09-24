// redis/internal/commands/strings.go
package commands

import (
	"redis/internal/resp"
	"redis/internal/storage"
)


func RegisterStringsCommand(r *Registry){
	r.Register("GET", getCommand)
	r.Register("SET", setCommand)
}

func getCommand(args []resp.Value, db storage.Engine) resp.Value{
	if len(args) != 1{
		return resp.Value{Type: "error", Str: "ERR wrong number of arguments for 'get' command"}
	}
	key := args[0].Bulk
	val, exists := db.Get(key)
	if !exists{
		return resp.Value{Type: "null"}
	}
	return resp.Value{Type: "bulk", Bulk: string(val.Data)}
}



func setCommand(args []resp.Value, db storage.Engine) resp.Value{
if len(args) < 2 {
		return resp.Value{Type: "error", Str: "ERR wrong number of arguments for 'set' command"}

	}

	key := args[0].Bulk
	value := []byte(args[1].Bulk)
	db.Set(key, value, 0)

return resp.Value{Type: "string", Str: "OK"}
}
