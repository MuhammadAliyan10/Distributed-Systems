package gateway

import (
	"fmt"
	"strings"
)

type ShardRoute struct {
	Address string
	CircuitBreaker *CircuitBreaker
}

type Router struct{
	routes map[string]*ShardRoute
}

func NewRouter() *Router{
	return &Router{
		routes: map[string]*ShardRoute{
			"shard_a": {Address: "http://localhost:8001", CircuitBreaker: NewCircuitBreaker()},
			"shard_b": {Address: "http://localhost:8002", CircuitBreaker: NewCircuitBreaker()},
		},
	}
}

func (r *Router) GetShard(user string) (*ShardRoute, error){
	if user == ""{
		return nil, fmt.Errorf("User cannot be empty")
	}
	firstChar := strings.ToUpper(string(user[0]))
	if firstChar >="A" && firstChar <= "M"{
		return  r.routes["shard_a"], nil
	}
	if firstChar >= "N" && firstChar <="Z"{
		return r.routes["shard_b"], nil
	}
return nil, fmt.Errorf("no shard found for user: %s", user)
}
