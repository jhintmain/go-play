package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)
import "github.com/gorilla/websocket"

var (
	upgrader            = websocket.Upgrader{}
	mu                  sync.Mutex
	roomClients         = make(map[string][]*websocket.Conn)
	roomClientsNickname = make(map[string][]string)
)

func handleUserList(w http.ResponseWriter, r *http.Request) {
	// 방 id
	roomID := r.URL.Query().Get("roomID")
	if roomID == "" {
		http.Error(w, "roomID is required", http.StatusBadRequest)
		return
	}

	nicknames, exist := roomClientsNickname[roomID]
	if !exist {
		http.Error(w, fmt.Sprintf("room %s not exist", roomID), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nicknames)
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// 방 id
	roomID := r.URL.Query().Get("roomID")
	if roomID == "" {
		http.Error(w, "roomId is required", http.StatusBadRequest)
		return
	}

	nickname := r.URL.Query().Get("nickname")
	if nickname == "" {
		http.Error(w, "nickname is required", http.StatusBadRequest)
		return
	}

	// 소켓 conn
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "upgrader err", http.StatusBadRequest)
		return
	}
	defer ws.Close()

	// 접속방에 client ws 정보 추가
	mu.Lock()
	roomClients[roomID] = append(roomClients[roomID], ws)
	roomClientsNickname[roomID] = append(roomClientsNickname[roomID], nickname)
	mu.Unlock()

	fmt.Printf("Client joined room [%s]\n", roomID)

	// ws 읽기 ( 무한 )
	for {
		_, message, err := ws.ReadMessage()
		// 연결 끊기면 해당방에서 ws제거
		if err != nil {
			mu.Lock()
			conns := roomClients[roomID]
			for i, conn := range conns {
				if conn == ws {
					roomClients[roomID] = append(conns[:i], conns[i+1:]...)
					removeNickname(roomID, nickname)
					break
				}
			}
			mu.Unlock()
		}

		// 같은방 ws들에게 메세지 전파
		mu.Lock()
		for _, conn := range roomClients[roomID] {
			// 나는 제외
			if conn != ws {
				conn.WriteMessage(websocket.TextMessage, message)
			}
		}
		mu.Unlock()
	}
}

func startServer() {
	http.HandleFunc("/ws", handleConnections)
	http.HandleFunc("/users", handleUserList)
	fmt.Println("Starting server... on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func removeNickname(roomID, nickname string) {
	nicknames := roomClientsNickname[roomID]
	for i, nick := range nicknames {
		if nickname == nick {
			roomClientsNickname[roomID] = append(nicknames[:i], nicknames[i+1:]...)
			break
		}
	}
}
