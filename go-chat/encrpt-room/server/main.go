package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"go-chat/encrpt-room/crypto"
	"log"
	"net/http"
	"net/url"
	"sync"
)
import "github.com/gorilla/websocket"

type ClientInfo struct {
	Conn     *websocket.Conn
	Nickname string
	PubKey   *ecdsa.PublicKey
}

type Room struct {
	Id      string
	PubKey  *ecdsa.PublicKey
	PriKey  *ecdsa.PrivateKey
	Clients []*ClientInfo
}

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // 모든 origin 허용
		},
	}
	mu    sync.Mutex
	rooms = make(map[string]*Room)
)

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
		http.Error(w, "publicKey is invalid 1", http.StatusBadRequest)
		return
	}

	pubKey, err := crypto.DecodePublicKey(pubKeyBytes)
	if err != nil {
		http.Error(w, "publicKey decode fail", http.StatusBadRequest)
		return
	}

	// 소켓 conn
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "upgrader err", http.StatusBadRequest)
		return
	}
	defer ws.Close()

	client := &ClientInfo{
		Conn:     ws,
		Nickname: nickname,
		PubKey:   pubKey,
	}
	// 접속방에 client ws 정보 추가
	mu.Lock()
	room, exist := rooms[roomID]
	if !exist {
		room = createRoom(roomID)
		rooms[roomID] = room
	}

	room.Clients = append(room.Clients, client)
	mu.Unlock()

	// 방 공유키 전달
	go sendEncryptedAESKey(client, room.PubKey)

	defer func() {
		leaveMsg := fmt.Sprintf("[%s]님이 퇴장하였습니다", nickname)
		broadcast(roomID, ws, []byte(leaveMsg))
	}()

	fmt.Printf("Client joined room [%s]\n", roomID)

	// 같은방 ws들에게 메세지 전파
	joinMessage := fmt.Sprintf("[%s] 님이 입장", nickname)
	broadcast(roomID, ws, []byte(joinMessage))

	// ws 읽기 ( 무한 )
	for {
		_, message, err := ws.ReadMessage()
		// 연결 끊기면 해당방에서 ws제거 & userlist 에서 제거
		if err != nil {
			log.Printf("read error from [%s]: %v", nickname, err)
			removeConnection(roomID, ws)
			return
		}

		sharedKey := crypto.GenerateSharedKey(pubKey, room.PriKey)
		decryptMessage, err := crypto.DecryptAES(sharedKey, message)
		if err != nil {
			log.Printf("decrypt message error: %v", err)
		}
		// 같은방 ws들에게 메세지 전파
		broadcast(roomID, ws, decryptMessage)
	}
}

func handleUserList(w http.ResponseWriter, r *http.Request) {
	// 방 id
	roomID := r.URL.Query().Get("roomID")
	if roomID == "" {
		http.Error(w, "roomID is required", http.StatusBadRequest)
		return
	}

	nicknames := []string{}
	room, exist := rooms[roomID]
	if !exist {
		http.Error(w, fmt.Sprintf("room %s not exist", roomID), http.StatusBadRequest)
		return
	}

	for _, client := range room.Clients {
		nicknames = append(nicknames, client.Nickname)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nicknames)
}

func createRoom(roomID string) *Room {
	keyPair := crypto.GenerateKey()
	room := &Room{
		Id:      roomID,
		PubKey:  keyPair.PubKey,
		PriKey:  keyPair.PriKey,
		Clients: []*ClientInfo{},
	}
	fmt.Printf("new room [%s]\n", roomID)
	return room
}

// 공개키 만들고 client들에게 보내기
func sendEncryptedAESKey(client *ClientInfo, pubKey *ecdsa.PublicKey) {
	pubKeyByte := elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)
	msg := map[string]interface{}{
		"type":   "key",
		"pubKey": url.QueryEscape(base64.StdEncoding.EncodeToString(pubKeyByte)),
	}

	fmt.Println(msg)

	client.Conn.WriteJSON(msg)
}

func StartServer() {
	http.HandleFunc("/ws", handleConnections)
	http.HandleFunc("/users", handleUserList)

	fmt.Println("Starting server... on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func removeConnection(roomID string, conn *websocket.Conn) {
	mu.Lock()
	defer mu.Unlock()

	// 소켓 연결 종료
	room := rooms[roomID]
	for i, client := range room.Clients {
		if client.Conn == conn {
			room.Clients = append(room.Clients[:i], room.Clients[i+1:]...)
			break
		}
	}

	if len(room.Clients) == 0 {
		delete(rooms, roomID)
	}
}

// broadcast
func broadcast(roomID string, sender *websocket.Conn, message []byte) {
	mu.Lock()
	defer mu.Unlock()

	room, exist := rooms[roomID]
	if !exist {
		fmt.Printf("Room [%s] is not exist", roomID)
		return
	}

	for _, client := range room.Clients {
		if client.Conn != sender {
			sharedKey := crypto.GenerateSharedKey(client.PubKey, room.PriKey)
			fmt.Printf("Shared key [%s] [%s]\n", client.Nickname, sharedKey)
			encryptMessage, err := crypto.EncryptAES(sharedKey, message)
			if err != nil {
				log.Printf("encrypt message error: %v", err)
			}
			fmt.Printf("Encrypted message [%s]\n", string(encryptMessage))
			if err := client.Conn.WriteMessage(websocket.TextMessage, encryptMessage); err != nil {
				log.Printf("write error to client [%s]: %v", client.Nickname, err)
				removeConnection(roomID, client.Conn)
			}
		}
	}
}
