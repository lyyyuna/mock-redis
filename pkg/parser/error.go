package parser

type ErrorProtocol struct {
	msg string
}

var _ error = (*ErrorProtocol)(nil)

func (e ErrorProtocol) Error() string {
	return "protocol error: " + e.msg
}
