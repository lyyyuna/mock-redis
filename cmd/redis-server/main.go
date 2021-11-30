package main

import (
	"errors"
	"sync"

	"github.com/lyyyuna/mock-redis/pkg/parser"
	"github.com/lyyyuna/mock-redis/pkg/server"
)

func main() {
	var mu sync.Mutex
	kvStore := make(map[string]string)

	redis := server.NewServer()
	redis.AddCommandHandler("set", func(conn *server.Conn, args []parser.Value) error {
		if len(args) != 3 {
			conn.WriteErrors(errors.New("ERR wrong number of arguments for 'set' command"))
		} else {
			mu.Lock()
			key := args[1].String()
			value := args[2].String()
			kvStore[key] = value
			mu.Unlock()

			conn.WriteSimpleStrings("OK")
		}

		return nil
	})

	redis.AddCommandHandler("get", func(conn *server.Conn, args []parser.Value) error {
		if len(args) != 2 {
			conn.WriteErrors(errors.New("ERR wrong number of arguments for 'get' command"))
		} else {
			mu.Lock()
			key := args[1].String()
			v, ok := kvStore[key]
			mu.Unlock()
			if !ok {
				conn.WriteSimpleStrings("")
			} else {
				conn.WriteBulkStrings([]byte(v))
			}
		}

		return nil
	})

	if err := redis.ListenAndServe(":6380"); err != nil {
		panic(err)
	}
}
