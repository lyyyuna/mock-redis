package server

import (
	"errors"
	"io"
	"net"
	"strings"
	"sync"

	"github.com/lyyyuna/mock-redis/pkg/parser"
)

type Conn struct {
	*parser.Reader
	*parser.Writer
	c net.Conn
}

func NewConn(conn net.Conn) *Conn {
	return &Conn{
		Reader: parser.NewReader(conn),
		Writer: parser.NewWriter(conn),
		c:      conn,
	}
}

type Server struct {
	mu       sync.Mutex
	handlers map[string]func(conn *Conn, args []parser.Value) error
}

func NewServer() *Server {
	return &Server{
		handlers: make(map[string]func(conn *Conn, args []parser.Value) error),
	}
}

func (s *Server) AddCommandHandler(command string, handler func(conn *Conn, args []parser.Value) error) {
	command = strings.ToUpper(command)
	s.mu.Lock()
	defer s.mu.Unlock()

	s.handlers[command] = handler
}

func (s *Server) ListenAndServe(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		go func() {
			defer conn.Close()

			err := s.handleConn(conn)
			if err != nil {
				io.WriteString(conn, "-ERR unknown error\r\n")
			}
		}()
	}
}

func (s *Server) handleConn(conn net.Conn) error {
	redisConn := NewConn(conn)

	for {
		v, err := redisConn.Read()
		if err != nil {
			return err
		}

		// Clients send commands to a Redis server as a RESP Array of Bulk Strings.
		values := v.Array()
		// empty
		if len(values) == 0 {
			continue
		}

		command := strings.ToUpper(values[0].String())

		s.mu.Lock()
		h := s.handlers[command]
		s.mu.Unlock()

		switch command {
		case "QUIT":
			redisConn.WriteSimpleStrings("OK")
			return nil
		case "PING":
			redisConn.WriteSimpleStrings("PONG")
			return nil
		}

		if h == nil {
			if err := redisConn.WriteErrors(errors.New("unknown command " + command)); err != nil {
				return err
			}
		} else {
			if err := h(redisConn, values); err != nil {
				return err
			}
		}
	}
}
