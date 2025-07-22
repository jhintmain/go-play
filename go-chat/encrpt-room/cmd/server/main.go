package main

import (
	"fmt"
	"go-chat/encrpt-room/internal/server"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/ws", server.HandleConnections)
	http.HandleFunc("/users", server.HandleUserList)

	fmt.Println("Starting server... on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
