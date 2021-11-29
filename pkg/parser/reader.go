package parser

import (
	"bufio"
	"io"
	"strconv"
)

// Reader is RESP Value reader from []byte
type Reader struct {
	rd *bufio.Reader
}

// NewReader creates a Reader from io
func NewReader(rd io.Reader) *Reader {
	return &Reader{
		rd: bufio.NewReader(rd),
	}
}

func (r *Reader) Read() (value Value, err error) {
	value, _, err = r.readValue()
	return
}

func (r *Reader) readValue() (val Value, n int, err error) {
	c, err := r.rd.ReadByte()
	if err != nil {
		return nullValue, 0, err
	}
	n++

	var rn int
	switch c {
	case '*':
		val, rn, err = r.readArrayValue()
	case '-':
		val, rn, err = r.readErrorsValue()
	case '+':
		val, rn, err = r.readSimpleValue()
	case ':':
		val, rn, err = r.readIntegersValue()
	case '$':
		val, rn, err = r.readBulkStringsValue()
	default:
		return nullValue, n, ErrorProtocol{"unknown first byte"}
	}

	n += rn
	// if err == io.EOF {
	// 	return nullValue, n, io.ErrUnexpectedEOF
	// }

	return val, n, err
}

// read until \r\n
func (r *Reader) readLine() (line []byte, n int, err error) {
	for {
		b, err := r.rd.ReadBytes('\n')
		if err != nil {
			return nil, 0, err
		}
		n += len(b)
		line = append(line, b...)

		if len(line) >= 2 && line[len(line)-2] == '\r' {
			break
		}
	}

	return line[:len(line)-2], n, nil
}

// readInt reads one int from one line
func (r *Reader) readInt() (int, int, error) {
	line, n, err := r.readLine()
	if err != nil {
		return 0, 0, err
	}

	i64, err := strconv.ParseInt(string(line), 10, 64)
	if err != nil {
		return 0, n, err
	}

	return int(i64), n, nil
}

func (r *Reader) readArrayValue() (val Value, n int, err error) {
	totalCnt, rn, err := r.readInt()
	n += rn
	if err != nil || totalCnt > 1024*1024 {
		return nullValue, n, err
	}

	// null array
	if totalCnt < 0 {
		return Value{typ: '*', null: true}, n, nil
	}

	arrvals := make([]Value, 0)
	for i := 0; i < totalCnt; i++ {
		val, rn, err := r.readValue()
		n += rn
		if err != nil {
			return nullValue, n, err
		}

		arrvals[i] = val
	}

	return Value{
		typ:   '*',
		array: arrvals,
	}, n, nil
}

//
func (r *Reader) readSimpleValue() (Value, int, error) {
	line, n, err := r.readLine()
	if err != nil {
		return nullValue, n, err
	}

	return Value{
		typ: '+',
		str: line,
	}, n, nil
}

func (r *Reader) readErrorsValue() (Value, int, error) {
	line, n, err := r.readLine()
	if err != nil {
		return nullValue, n, err
	}

	return Value{
		typ: '-',
		str: line,
	}, n, nil
}

func (r *Reader) readIntegersValue() (Value, int, error) {
	integer, n, err := r.readInt()
	if err != nil {
		return nullValue, n, err
	}

	return Value{
		typ:     ':',
		integer: integer,
	}, n, nil
}

func (r *Reader) readBulkStringsValue() (val Value, n int, err error) {
	strLen, rn, err := r.readInt()
	n += rn
	if err != nil {
		return nullValue, n, err
	}

	if strLen < 0 {
		return Value{
			typ:  '$',
			null: true,
		}, n, nil
	}

	if strLen > 512*1024*1024 {
		return nullValue, n, ErrorProtocol{"invalid bulk length"}
	}

	bulk := make([]byte, strLen+2)
	rn, err = io.ReadFull(r.rd, bulk)
	n += rn
	if err != nil {
		return nullValue, n, err
	}

	if bulk[strLen] != '\r' || bulk[strLen+1] != '\n' {
		return nullValue, n, ErrorProtocol{"invalid bulk string ending"}
	}

	return Value{
		typ: '$',
		str: bulk[:strLen],
	}, n, nil
}
