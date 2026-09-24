// redis/internal/commands/strings.go
package commands

import (
	"redis/internal/resp"
	"redis/internal/storage"
	"strconv"
	"strings"
	"time"
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

	var ttl time.Duration = 0

	if len(args) >= 4 {
		modifier := strings.ToUpper(args[2].Bulk)

		if modifier == "EX"{
			seconds, err := strconv.Atoi(args[3].Bulk)
			if err != nil{
				return resp.Value{Type: "error", Str: "ERR value is not an integer or out of range"}
			}
			ttl = time.Duration(seconds) * time.Second
		}

	}
	db.Set(key, value, ttl)

return resp.Value{Type: "string", Str: "OK"}
}
