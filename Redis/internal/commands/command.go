// redis/internal/commands/command.go
package commands

import (
	"redis/internal/resp"
	"redis/internal/storage"
	"strings"
)

type Handler func(args []resp.Value, db storage.Engine) resp.Value


type Registry struct{
	handlers map[string]Handler
}


func NewRegistry() *Registry{
	return &Registry{
		handlers: make(map[string]Handler),
	}
}

func (r *Registry) Register(name string, handler Handler){
	r.handlers[strings.ToUpper(name)] = handler
}


func (r *Registry) Execute(cmdName string, args []resp.Value, db storage.Engine) resp.Value{
	handler, exists := r.handlers[strings.ToUpper(cmdName)]

	if !exists{
		return resp.Value{Type: "error", Str: "ERR unknown command '" + cmdName + "'" }
	}
	return handler(args, db)
}
