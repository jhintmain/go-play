package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"fmt"
	"github.com/gorilla/websocket"
	"go-chat/encrpt-room/internal/crypto"
	"go-chat/encrpt-room/internal/model"
	"log"
	"net/url"
)

func createRoom(roomID string) *model.Room {
	keyPair := crypto.GenerateKey()
	room := &model.Room{
		Id:      roomID,
		PubKey:  keyPair.PubKey,
		PriKey:  keyPair.PriKey,
		Clients: make(map[string]model.ClientInterface),
	}
	fmt.Printf("new room [%s]\n", roomID)
	return room
}

// 공개키 만들고 client들에게 보내기
func sendEncryptedAESKey(client *model.Client, pubKey *ecdsa.PublicKey) {
	pubKeyByte := elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)
	msg := map[string]interface{}{
		"type":   "key",
		"pubKey": url.QueryEscape(base64.StdEncoding.EncodeToString(pubKeyByte)),
	}

	client.Conn.WriteJSON(msg)
}

func removeConnection(roomID, clientID string) {
	mu.Lock()
	defer mu.Unlock()

	// 소켓 연결 종료
	room, exist := rooms[roomID]
	if !exist {
		return
	}

	_, exist = room.Clients[clientID]
	if !exist {
		return
	}

	delete(room.Clients, clientID)

	if len(room.Clients) == 0 {
		delete(rooms, roomID)
	}
}

func broadcast(roomID, senderID string, message []byte) {
	mu.Lock()
	defer mu.Unlock()

	room, exist := rooms[roomID]
	if !exist {
		fmt.Printf("Room [%s] is not exist", roomID)
		return
	}

	_, exist = room.Clients[senderID]
	if !exist {
		fmt.Printf("client [%s] is not exist", senderID)
		return
	}

	for _, client := range room.Clients {
		var receiverID = client.GetID()

		if receiverID != senderID {
			sharedKey := crypto.GenerateSharedKey(client.GetPubKey(), room.PriKey)
			encryptMessage, err := crypto.EncryptAES(sharedKey, message)
			if err != nil {
				log.Printf("encrypt message error: %v", err)
			}
			fmt.Printf("Encrypted message [%s]\n", string(encryptMessage))

			if err := client.GetConn().WriteMessage(websocket.TextMessage, encryptMessage); err != nil {
				log.Printf("write error to client [%s]: %v", client.GetNickname(), err)
				removeConnection(roomID, client.GetID())
			}
		}
	}
}
