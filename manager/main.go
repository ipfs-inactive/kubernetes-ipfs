package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
)

var (
	optPort = flag.Int("port", 9232, "Port")

func main() {
	
}

func server() {
	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mngr := NewManager(mainCtx)
	server := rpc.NewServer()
	err := server.Register(mngr)
	if err != nil {
		log.Fatal("register:", err)
	}

	server.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)
	l, err := net.Listen("tcp", ":9232")
	if err != nil {
		log.Fatal("listen:", err)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal("accpet:", err)
		}

		go server.ServeCodec(jsonrpc.NewServerCodec(conn))
	}
}
