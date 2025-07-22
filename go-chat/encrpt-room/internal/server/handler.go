package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go-chat/encrpt-room/internal/crypto"
	"go-chat/encrpt-room/internal/model"
	"log"
	"net/http"
	"sync"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // 모든 origin 허용
		},
	}
	mu    sync.Mutex
	rooms = make(map[string]*model.Room)
)

func HandleConnections(w http.ResponseWriter, r *http.Request) {
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
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "upgrader err", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	client := &model.Client{
		ID:       uuid.NewString(),
		Conn:     conn,
		Nickname: nickname,
		PubKey:   pubKey,
		SendChan: make(chan []byte, 256),
	}
	var cInterface model.ClientInterface = client
	// 접속방에 client conn 정보 추가
	mu.Lock()
	room, exist := rooms[roomID]
	if !exist {
		room = createRoom(roomID)
		rooms[roomID] = room
	}

	room.Clients[client.ID] = cInterface
	mu.Unlock()

	// 방 공개키 전달
	go sendEncryptedAESKey(client, room.PubKey)

	defer func() {
		leaveMsg := fmt.Sprintf("[%s]님이 퇴장하였습니다", nickname)
		broadcast(roomID, client.ID, []byte(leaveMsg))
	}()

	fmt.Printf("Client joined room [%s]\n", roomID)

	// 같은방 ws들에게 메세지 전파
	joinMessage := fmt.Sprintf("[%s] 님이 입장", nickname)
	broadcast(roomID, client.ID, []byte(joinMessage))

	// ws 읽기 ( 무한 )
	for {
		_, message, err := conn.ReadMessage()
		// 연결 끊기면 해당방에서 ws제거 & userlist 에서 제거
		if err != nil {
			log.Printf("read error from [%s]: %v", nickname, err)
			removeConnection(roomID, client.ID)
			return
		}

		// client의 공개키와 방의 비밀키로 공유키 생성
		sharedKey := crypto.GenerateSharedKey(pubKey, room.PriKey)
		// 공유키로 메세지 복호화
		decryptMessage, err := crypto.DecryptAES(sharedKey, message)
		if err != nil {
			log.Printf("decrypt message error: %v", err)
		}
		// 같은방 ws들에게 메세지 전파
		broadcast(roomID, client.ID, decryptMessage)
	}
}

func HandleUserList(w http.ResponseWriter, r *http.Request) {
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
		nicknames = append(nicknames, client.GetNickname())
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nicknames)
}
