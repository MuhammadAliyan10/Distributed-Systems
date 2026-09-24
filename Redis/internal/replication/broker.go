package replication

import (
	"net"
	"sync"
)


type Broker struct{
	mu sync.Mutex
	replicas map[net.Conn]bool
}


func NewBroker() *Broker{
	return &Broker{
		replicas: make(map[net.Conn]bool),
	}
}

func (b *Broker) AddReplica(conn net.Conn){
	b.mu.Lock()
	defer b.mu.Unlock()

	b.replicas[conn] = true

}

func (b *Broker) Broadcast(rawCommand []byte) {
b.mu.Lock()
defer b.mu.Unlock()

for conn := range b.replicas{
	_, err := conn.Write(rawCommand)
	if err != nil{
		delete(b.replicas, conn)
		conn.Close()
	}
}
}
