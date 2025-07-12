package main

import (
	"bufio"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"os"
)

func startClient() {
	fmt.Println("Starting client... on :8080")

	// 접속할 roomID 입력받기
	fmt.Print("Enter room ID:")
	var roomID string
	fmt.Scanln(&roomID)

	// roomID 소켓 연결
	url := fmt.Sprintf("ws://localhost:8080/ws?roomID=%s", roomID)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer conn.Close()

	// ws write 읽기 go 루틴으로 띄어두기
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			fmt.Printf("Message Received: %s\n", message)
		}
	}()

	fmt.Println("Start chatting")

	// 사용자 입력 받기
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		// 입력받은 text ws에 써주기
		text := scanner.Text()
		err := conn.WriteMessage(websocket.TextMessage, []byte(text))
		if err != nil {
			log.Println("write:", err)
			return
		}
	}
}
