package aof

import (
	"os"
	"sync"
	"time"
)

type AOF struct{
	file *os.File
	mu sync.Mutex
	closeChan chan struct{}
	wg sync.WaitGroup
}

func NewAOF(path string) (*AOF, error){
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil{
		return nil, err
	}
 a := &AOF{
	file: f,
	closeChan: make(chan struct{}),
 }
 a.wg.Add(1)

 go a.fsyncWorker()

 return a, nil
}

func (a *AOF) fsyncWorker(){


	defer a.wg.Done()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for{
		select {
		case <-ticker.C:
			a.mu.Lock()
			a.file.Sync()
			a.mu.Unlock()
		case <-a.closeChan:
			return
	}
}

}



func (a *AOF) Write(value []byte) error{
a.mu.Lock()
defer a.mu.Unlock()

_, err := a.file.Write(value)
if err != nil {
	return  err
}
return nil

}

func (a *AOF) Close() error{
	close(a.closeChan)

	a.wg.Wait()

	a.mu.Lock()
	defer a.mu.Unlock()

	a.file.Sync()

	return  a.file.Close()
}
