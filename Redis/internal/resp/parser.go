// redis/internal/resp/parser.go
package resp

import (
	"bufio"
	"errors"
	"io"
	"strconv"
)

type Value struct{
	Type  string
	Str   string
	Num   int
	Bulk  string
	Array []Value
}

type Parser struct{
	reader *bufio.Reader
}

func NewParser(r io.Reader) *Parser{
	return &Parser{reader:bufio.NewReader(r)}
}



func (p *Parser) Read() (Value, error){
_type, err := p.reader.ReadByte()
if err != nil{
	return Value{}, err
}
switch _type {
case '*':
	return p.readArray()

case '$':
	return p.readBulkString()

default:
	return Value{}, errors.New("unknown RESP type")
}

}

func (p *Parser) readLine() (line []byte, err error){
	for {
		b, err := p.reader.ReadByte()
		if err != nil{
			return nil, err
	}
	line = append(line, b)
	if len(line) >=2 && line[len(line)-2] == '\r'{
		break
	}
}

return line[:len(line)-2], nil
}

func (p *Parser) readArray()(Value, error){
	line, err:= p.readLine()
	if err != nil {
		return Value{},err
	}

	count, _ := strconv.Atoi(string(line))

	val := Value{
		Type: "array",
		Array: make([]Value, count),

	}

	for i:=0; i<count; i++{
		val.Array[i], err = p.Read()
		if err != nil {
			return Value{}, err
		}
	}

	return val, nil



}

func (p *Parser) readBulkString()(Value, error){
line, err:= p.readLine()
if err != nil {
		return Value{}, err
	}

	length, _ := strconv.Atoi(string(line))

	bulk := make([]byte, length)
_, err = io.ReadFull(p.reader, bulk)
if err != nil{
	return Value{}, err
}
	p.readLine()

		return Value{
		Type: "bulk",
		Bulk: string(bulk),
	}, nil
}
