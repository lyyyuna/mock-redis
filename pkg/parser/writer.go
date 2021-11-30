package parser

import "io"

type Writer struct {
	w io.Writer
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w: w,
	}
}

func (w *Writer) WriteValue(v Value) error {
	b, err := v.MarshalRESP()
	if err != nil {
		return err
	}

	_, err = w.w.Write(b)

	return err
}

func (w *Writer) WriteSimpleStrings(s string) error {
	return w.WriteValue(simpleStringsValue(s))
}

func (w *Writer) WriteBulkStrings(b []byte) error {
	return w.WriteValue(bulkStringsValue(b))
}

func (w *Writer) WriteErrors(err error) error {
	return w.WriteValue(errorsValue(err))
}

func (w *Writer) WriteIntegers(i int) error {
	return w.WriteValue(integersValue(i))
}

func (w *Writer) WriteArrays(vals []Value) error {
	return w.WriteValue(arrayValue(vals))
}
