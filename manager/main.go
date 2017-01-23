package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"

	stump "github.com/whyrusleeping/stump"
)

var (
	optAddress = flag.String("address", "localhost:9232", "Address to listen or connect to.")
)

func main() {
	stump.Verbose = true
	flag.Parse()

	switch flag.Arg(0) {
	case "server":
		log.Printf("Running server at: %s", *optAddress)
		server()
	case "client":
		client(flag.Args()[1:])
	default:
		log.Fatalf("Invalid argument: '%s'", flag.Arg(0))
	}
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
	l, err := net.Listen("tcp", *optAddress)
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

func client(args []string) {
	if len(args) == 0 {
		log.Fatal("not enough arguments")
	}

	log.Printf("Connecting to RPC server at: %s", *optAddress)
	conn, err := net.Dial("tcp", *optAddress)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer conn.Close()
	c := jsonrpc.NewClient(conn)

	empty := struct{}{}

	switch args[0] {
	case "stop":
		status := -1
		err = c.Call("Manager.Stop", empty, &status)
		defer fmt.Println(status)
	case "start":
		startArgs := args[1:]
		err = c.Call("Manager.Start", startArgs, &empty)
	case "version":
		if len(args) != 2 {
			log.Fatal("wrong args length")
		}
		ver := args[1]
		err = c.Call("Manager.ChangeVersion", ver, &empty)
	default:
		log.Fatalf("Unknown function: %s", args[0])
	}

	if err != nil {
		log.Fatal("call: ", err)
	}

}
