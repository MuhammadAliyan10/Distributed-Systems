package commands

import (
	"redis/internal/resp"
	"redis/internal/storage"
)

func RegisterKeyCommands(r *Registry) {
	r.Register("DEL", delCommand)
	r.Register("PING", pingCommand)
}

func delCommand(args []resp.Value, db storage.Engine) resp.Value {
	if len(args) < 1 {
		return resp.Value{Type: "error", Str: "ERR wrong number of arguments for 'del' command"}
	}

	deletedCount := 0
	for _, arg := range args {
		deletedCount += db.Del(arg.Bulk)
	}

	return resp.Value{Type: "integer", Num: deletedCount}
}

func pingCommand(args []resp.Value, db storage.Engine) resp.Value{
return resp.Value{Type: "string", Str: "PONG"}
}
