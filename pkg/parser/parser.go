package parser

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

// Type represents a Value type
type Type byte

const (
	SimpleStrings Type = '+'
	Errors        Type = '-'
	Integers      Type = ':'
	BulkStrings   Type = '$'
	Arrays        Type = '*'
)

func (t Type) String() string {
	switch t {
	case SimpleStrings:
		return "SimpleStrings"
	case Errors:
		return "Errors"
	case Integers:
		return "Integers"
	case BulkStrings:
		return "BulkStrings"
	case Arrays:
		return "Arrays"
	default:
		return "Unknown"
	}
}

// Value represents the RESP data type
//
// It stores all possible value.
type Value struct {
	typ     Type
	integer int
	str     []byte
	array   []Value
	null    bool
}

func (v Value) String() string {
	switch v.typ {
	case BulkStrings, SimpleStrings, Errors:
		return string(v.str)
	case Integers:
		return strconv.FormatInt(int64(v.integer), 10)
	case Arrays:
		return fmt.Sprintf("%v", v.array)
	}

	return ""
}

func (v Value) Integer() int {
	switch v.typ {
	case Integers:
		return v.integer
	default:
		n, _ := strconv.ParseInt(v.String(), 10, 64)
		return int(n)
	}
}

func (v Value) Bytes() []byte {
	switch v.typ {
	case BulkStrings, SimpleStrings, Errors:
		return v.str
	default:
		return []byte(v.String())
	}
}

func (v Value) IsNull() bool {
	return v.null
}

func (v Value) Error() error {
	switch v.typ {
	case Errors:
		return errors.New(string(v.str))
	}

	return nil
}

func (v Value) Array() []Value {
	if !v.null && v.typ == Arrays {
		return v.array
	}

	return nil
}

var nullValue = Value{null: true}

func replaceNewlineWithSpace(oriS string) string {
	re := regexp.MustCompile("\r")
	oriS = re.ReplaceAllString(oriS, " ")
	re = regexp.MustCompile("\n")
	newS := re.ReplaceAllString(oriS, " ")

	return newS
}

func simpleStringsValue(s string) Value {
	return Value{
		typ: SimpleStrings,
		str: []byte(replaceNewlineWithSpace(s)),
	}
}

func bulkStringsValue(b []byte) Value {
	return Value{
		typ: BulkStrings,
		str: b,
	}
}

func nullsValue() Value {
	return Value{
		typ:  BulkStrings,
		null: true,
	}
}

func errorsValue(err error) Value {
	if err == nil {
		return Value{typ: Errors}
	}

	return Value{
		typ: Errors,
		str: []byte(err.Error()),
	}
}

func integersValue(i int) Value {
	return Value{
		typ:     Integers,
		integer: i,
	}
}

func arrayValue(vals []Value) Value {
	return Value{
		typ:   Arrays,
		array: vals,
	}
}

func (v Value) MarshalRESP() ([]byte, error) {
	return marshalRESP(v)
}

func marshalRESP(v Value) ([]byte, error) {
	switch v.typ {
	case SimpleStrings:
		return marshalSimpleStringsRESP(v)
	case Errors:
		return marshalErrorsRESP(v)
	case Integers:
		return marshalIntegersRESP(v)
	case BulkStrings:
		return marshalBulkStringsRESP(v)
	case Arrays:
		return marshalArraysRESP(v)
	default:
		if v.null {
			return []byte("$-1\r\n"), nil
		}
		return nil, errors.New("unknown resp type")
	}
}

func marshalSimpleStringsRESP(v Value) ([]byte, error) {
	b := v.str
	bb := make([]byte, 3+len(b))
	bb[0] = byte(SimpleStrings)
	copy(bb[1:], b)
	bb[1+len(b)] = '\r'
	bb[1+len(b)+1] = '\n'

	return bb, nil
}

func marshalErrorsRESP(v Value) ([]byte, error) {
	b := v.str
	bb := make([]byte, 3+len(b))
	bb[0] = byte(Errors)
	copy(bb[1:], b)
	bb[1+len(b)] = '\r'
	bb[1+len(b)+1] = '\n'

	return bb, nil
}

func marshalIntegersRESP(v Value) ([]byte, error) {
	b := []byte(strconv.FormatInt(int64(v.integer), 10))
	bb := make([]byte, 3+len(b))
	bb[0] = byte(Integers)
	copy(bb[1:], b)
	bb[1+len(b)] = '\r'
	bb[1+len(b)+1] = '\n'

	return bb, nil
}

func marshalBulkStringsRESP(v Value) ([]byte, error) {
	if v.null {
		return []byte("$-1\r\n"), nil
	}

	strLen := []byte(strconv.FormatInt(int64(len(v.str)), 10))
	bb := make([]byte, 1+len(strLen)+2+len(v.str)+2)

	// "$6\r\nfoobar\r\n"
	bb[0] = byte(BulkStrings)
	copy(bb[1:], strLen)
	bb[1+len(strLen)] = '\r'
	bb[1+len(strLen)+1] = '\n'

	copy(bb[1+len(strLen)+2:], v.str)
	bb[1+len(strLen)+2+len(v.str)] = '\r'
	bb[1+len(strLen)+2+len(v.str)+1] = '\n'

	return bb, nil
}

func marshalArraysRESP(v Value) ([]byte, error) {
	if v.null {
		return []byte("*-1\r\n"), nil
	}

	arrLen := []byte(strconv.FormatInt(int64(len(v.array)), 10))

	var buf bytes.Buffer
	buf.WriteByte(byte(Arrays))
	buf.Write(arrLen)
	buf.Write([]byte("\r\n"))

	for i := 0; i < len(v.array); i++ {
		data, err := v.array[i].MarshalRESP()
		if err != nil {
			return nil, err
		}
		buf.Write(data)
	}

	return buf.Bytes(), nil
}
