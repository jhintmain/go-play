package client

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"go-chat/encrpt-room/crypto"
	"io"
	"log"
	"net/url"
	"os"
)

var aesKey []byte

func StartClient() {
	fmt.Println("Starting client... on :8080")

	// 접속할 roomID 입력받기
	fmt.Print("Enter room ID:")
	var roomID string
	fmt.Scanln(&roomID)

	fmt.Print("What is nickname:")
	var nickname string
	fmt.Scanln(&nickname)

	keyPair := crypto.GenerateKey()
	pubKey := keyPair.GetPubKeyToString()

	log.Printf("generated pubKey: %s", pubKey)
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
	defer conn.Close()

	// ws write 읽기 go 루틴으로 띄어두기
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}

			var data map[string]string

			if err := json.Unmarshal(message, &data); err == nil && data["type"] == "key" {
				// 서버로 부터 받은 방 공개키
				fmt.Printf("pubKey: %s\n", data["pubKey"])
				decodedStr, err := url.QueryUnescape(data["pubKey"])
				if err != nil {
					log.Fatalf("url decode error: %v", err)
				}
				roomPubKeyByte, err := base64.StdEncoding.DecodeString(decodedStr)
				if err != nil {
					log.Fatalf("base64 decode error: %v", err)
				}
				roomPubKey, err := crypto.DecodePublicKey(roomPubKeyByte)
				if err != nil {
					log.Fatalf("decode public key error: %v", err)
				}

				//fmt.Printf("roomPubKey: %s", roomPubKey)

				// 공유키 생성
				sharedKey := crypto.GenerateSharedKey(roomPubKey, keyPair.PriKey)
				//fmt.Printf("sharedKey: %s", sharedKey)

				// 대칭키 생성( 공유키가 곧 대칭키)
				aesKey = sharedKey

				continue
			}

			if aesKey != nil {
				decryptedMessage, err := crypto.DecryptAES(aesKey, message)
				if err != nil {
					fmt.Sprintf("decrypt error:%v", err)
				} else {
					fmt.Printf("message: %s\n", message)
					fmt.Printf("%s\n", string(decryptedMessage))
				}
			} else {
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
		encryptedMessage, _ := crypto.EncryptAES(aesKey, []byte(plain))
		if err := conn.WriteMessage(websocket.TextMessage, encryptedMessage); err != nil {
			log.Println("write:", err)
			return
		}
	}
}
