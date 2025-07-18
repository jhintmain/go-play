package main

import (
	"go-chat/encrpt-room/client"
	"go-chat/encrpt-room/server"
	"os"
)

func main() {

	if len(os.Args) != 2 {
		println("Usage: go run main.go server | client")
		return
	}

	if os.Args[1] == "server" {
		server.StartServer()
	} else if os.Args[1] == "client" {
		client.StartClient()
	}
}
