package client

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"go-chat/encrpt-room/internal/crypto"
	"io"
	"log"
	"net/url"
	"os"
)

var sharedKey []byte

func Start() {
	// 접속할 roomID 입력받기
	fmt.Print("Enter room ID:")
	var roomID string
	fmt.Scanln(&roomID)

	fmt.Print("What is nickname:")
	var nickname string
	fmt.Scanln(&nickname)

	keyPair := crypto.GenerateKey()
	pubKey := keyPair.GetPubKeyToString()

	log.Printf("generated keypair -- ok")

	// roomID 소켓 연결
	serverUrl := fmt.Sprintf("ws://localhost:8080/ws?roomID=%s&nickname=%s&pubKey=%s", roomID, nickname, pubKey)
	conn, resp, err := websocket.DefaultDialer.Dial(serverUrl, nil)
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			log.Fatalf("dial error: %v\nstatus: %s\nbody: %s", err, resp.Status, string(body))
		} else {
			log.Fatal("dial error:", err)
		}
	}
	log.Printf("connected to server -- ok")
	defer conn.Close()

	// ws write 읽기 go 루틴으로 띄어두기 > 메세지는 들어오는 대로 읽어야 하니까
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Fatalf("message read error : %v", err)
				return
			}

			var data map[string]string

			if err := json.Unmarshal(message, &data); err == nil && data["type"] == "key" {
				// 서버로 부터 받은 방 공개키
				decodedStr, err := url.QueryUnescape(data["pubKey"])
				if err != nil {
					log.Fatalf("decode pubKey error: %v", err)
				}
				baseDecodePubKey, err := base64.StdEncoding.DecodeString(decodedStr)
				if err != nil {
					log.Fatalf("base64 decode error: %v", err)
				}
				roomPubKey, err := crypto.DecodePublicKey(baseDecodePubKey)
				if err != nil {
					log.Fatalf("decode public key error: %v", err)
				}
				// 공유키 생성
				sharedKey = crypto.GenerateSharedKey(roomPubKey, keyPair.PriKey)

				continue
			}

			if sharedKey != nil {
				decryptedMessage, err := crypto.DecryptAES(sharedKey, message)
				if err != nil {
					log.Printf("decrypt error: %v", err)
				} else {
					// client에 복호화된 메세지 출력
					fmt.Printf("%s\n", string(decryptedMessage))
				}
			} else {
				// client에 복호화 되지 않은 메세지 출ㄹ격
				fmt.Printf("(no ecrypted message) %s\n", string(message))
			}
		}
	}()

	// 사용자 입력 받기
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		// 입력받은 text ws에 써주기
		msg := scanner.Text()
		plain := fmt.Sprintf("%s : %s", nickname, msg)
		encryptedMessage, _ := crypto.EncryptAES(sharedKey, []byte(plain))
		if err := conn.WriteMessage(websocket.TextMessage, encryptedMessage); err != nil {
			log.Fatalf("write encrypted message error: %v", err)
		}
	}
}
