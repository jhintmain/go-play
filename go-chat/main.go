package main

import "os"

func main() {

	if len(os.Args) != 2 {
		println("Usage: go run main.go server | client")
		return
	}

	if os.Args[1] == "server" {
		startServer()
	} else if os.Args[1] == "client" {
		startClient()
	}
}
