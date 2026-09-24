package aof

import (
	"os"
	"redis/internal/storage"
	"strconv"
)



func formatSetCommand(key string, value []byte) []byte{
	var result []byte

	result = append(result, '*')
	result = append(result, '3', '\r', '\n')

	result = append(result, '$', '3', '\r', '\n')
	result = append(result, "SET"...)
	result = append(result, '\r', '\n')

result = append(result, '$')
	result = append(result, strconv.Itoa(len(key))...)
	result = append(result, '\r', '\n')
	result = append(result, key...)
	result = append(result, '\r', '\n')

	result = append(result, '$')
	result = append(result, strconv.Itoa(len(value))...)
	result = append(result, '\r', '\n')
	result = append(result, value...)
	result = append(result, '\r', '\n')
	return result
}


func (a *AOF) Rewrite(db storage.Engine) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	tempFileName := a.file.Name() + ".temp"
	tempFile, err := os.OpenFile(tempFileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		 return err
	}
	db.Iterate(func(key string, val storage.Value){
	rawBytes := formatSetCommand(key, val.Data)
	tempFile.Write(rawBytes)

	})

	tempFile.Sync()
	tempFile.Close()

	a.file.Close()
	os.Rename(tempFileName, a.file.Name())

		newFile, _ := os.OpenFile(a.file.Name(), os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	a.file = newFile
	return nil
}
