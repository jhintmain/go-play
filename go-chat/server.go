package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)
import "github.com/gorilla/websocket"

type ClientInfo struct {
	Conn      *websocket.Conn
	Nickname  string
	PublicKey *ecdsa.PublicKey
}

var (
	upgrader            = websocket.Upgrader{}
	mu                  sync.Mutex
	roomClients         = make(map[string][]*ClientInfo)
	roomClientsNickname = make(map[string][]string)
)

func handleUserList(w http.ResponseWriter, r *http.Request) {
	// 방 id
	roomID := r.URL.Query().Get("roomID")
	if roomID == "" {
		http.Error(w, "roomID is required", http.StatusBadRequest)
		return
	}

	nicknames := []string{}
	clients, exist := roomClients[roomID]
	if !exist {
		http.Error(w, fmt.Sprintf("room %s not exist", roomID), http.StatusBadRequest)
		return
	}

	for _, client := range clients {
		nicknames = append(nicknames, client.Nickname)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nicknames)
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// 방 id
	roomID := r.URL.Query().Get("roomID")
	nickname := r.URL.Query().Get("nickname")
	pubKeyBase64 := r.URL.Query().Get("pubKey")
	if roomID == "" || nickname == "" || pubKeyBase64 == "" {
		http.Error(w, "roomID , nickname , publicKey is required", http.StatusBadRequest)
		return
	}

	pubKeyBytes, err := base64.StdEncoding.DecodeString(pubKeyBase64)
	if err != nil {
		http.Error(w, "publicKey is invalid", http.StatusBadRequest)
		return
	}

	publicKey, err := decodePublicKey(pubKeyBytes)
	if err != nil {
		http.Error(w, "publicKey is invalid", http.StatusBadRequest)
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
	client := &ClientInfo{
		Conn:      ws,
		Nickname:  nickname,
		PublicKey: publicKey,
	}
	log.Printf("new client: %v", client)
	roomClients[roomID] = append(roomClients[roomID], client)
	mu.Unlock()

	fmt.Printf("Client joined room [%s]\n", roomID)

	// 같은방 ws들에게 메세지 전파
	joinMessage := fmt.Sprintf("[%s] 님이 입장", nickname)
	broadcast(roomID, ws, []byte(joinMessage))

	// ws 읽기 ( 무한 )
	for {
		_, message, err := ws.ReadMessage()
		// 연결 끊기면 해당방에서 ws제거 & userlist 에서 제거
		if err != nil {
			removeConnection(roomID, nickname, ws)
		}
		// 같은방 ws들에게 메세지 전파
		broadcast(roomID, ws, message)
	}
}

func startServer() {
	http.HandleFunc("/ws", handleConnections)
	http.HandleFunc("/users", handleUserList)

	fmt.Println("Starting server... on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func removeConnection(roomID, nickname string, conn *websocket.Conn) {
	mu.Lock()
	defer mu.Unlock()

	// 소켓 연결 종료
	clients := roomClients[roomID]
	for i, client := range clients {
		if client.Conn == conn {
			roomClients[roomID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}
}

// broadcast
func broadcast(roomID string, sender *websocket.Conn, message []byte) {
	mu.Lock()
	defer mu.Unlock()

	clients := roomClients[roomID]
	for _, client := range clients {
		if client.Conn != sender {
			client.Conn.WriteMessage(websocket.TextMessage, message)
		}
	}
}

func decodePublicKey(pubKeyBytes []byte) (*ecdsa.PublicKey, error) {
	x, y := elliptic.Unmarshal(elliptic.P256(), pubKeyBytes)
	if x == nil || y == nil {
		return nil, fmt.Errorf("invalid public key")
	}

	return &ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}, nil
}
