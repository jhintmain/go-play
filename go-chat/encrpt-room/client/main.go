package client

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"go-chat/encrpt-room/crypto"
	"log"
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

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal("GenerateKey key fail", err)
	}
	pubKey := &priv.PublicKey

	pubKeyBytes := elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)

	pubKeyBase64 := base64.StdEncoding.EncodeToString(pubKeyBytes)

	log.Printf("pubKeyBase64: %s", pubKeyBase64)
	// roomID 소켓 연결
	url := fmt.Sprintf("ws://localhost:8080/ws?roomID=%s&nickname=%s&pubKey=%s", roomID, nickname, pubKeyBase64)
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

			var data map[string]string

			if json.Unmarshal(message, &data) != nil && data["type"] == "key" {
				// 서버로 부터 받은 공유키 decode
				encryptedSharedKey, _ := base64.StdEncoding.DecodeString(data["crypt_key"])

				// decode 된 공유키 내 공유/비밀키로 대칭키 생성 > 내 key가지고 암호화되서 내껄로 하면됨
				sharedKey := crypto.GenerateSharedKey(pubKey, priv.D.Bytes())

				// 대칭키로 암호화된 key 최종 복호화
				aesKey, _ = crypto.DecryptAES(sharedKey, encryptedSharedKey)
				fmt.Println("received AES key:", aesKey)
				continue
			}

			if aesKey != nil {
				decryptedMessage, err := crypto.DecryptAES(aesKey, message)
				if err != nil {
					fmt.Sprintf("decrypt error:%v", err)
				} else {
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
