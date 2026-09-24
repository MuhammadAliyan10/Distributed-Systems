// redis/internal/resp/writer.go
package resp

import (
	"io"
	"strconv"
)


type Writer struct{
	writer io.Writer
}

func NewWriter(w io.Writer) *Writer{
	return &Writer{writer: w}
}


func (w *Writer) WriteSimpleString(s string) error{
	msg := "+" + s + "\r\n"
	_, err := w.writer.Write([]byte(msg))
	return err
}

func (w *Writer) WriteError(e string) error{
	msg := "-" + e + "\r\n"
	_, err := w.writer.Write([]byte(msg))
	return err
}

func (w *Writer) WriteBulkString(bulk string) error{
	length := strconv.Itoa(len(bulk))
	msg := "$" + length + "\r\n" + bulk + "\r\n"
	_, err := w.writer.Write([]byte(msg))
	return err
}


func (w *Writer) WriteNull() error{
_, err := w.writer.Write([]byte("$-1\r\n"))
return err
}


func (w *Writer) WriteInteger(n int) error {
	num := strconv.Itoa(n)
	msg := ":" + num + "\r\n"
	_, err := w.writer.Write([]byte(msg))
	return  err
}
