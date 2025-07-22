package main

import (
	"fmt"
	"go-chat/encrpt-room/internal/client"
)

func main() {
	fmt.Println("Starting client... on :8080")

	client.Start()
}
