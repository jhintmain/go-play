package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
)

// 공유키 & 비밀키로 복호화 할수있는 대칭키 생성
func GenerateSharedKey(pubKey *ecdsa.PublicKey, k []byte) []byte {
	sharedX, _ := pubKey.Curve.ScalarMult(pubKey.X, pubKey.Y, k)
	sharedKey := sharedX.Bytes()
	if len(sharedKey) > 16 {
		sharedKey = sharedKey[0:16]
	}
	return sharedKey
}

// AES CBC 복호화
func DecryptAES(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]
	plaintext := make([]byte, len(ciphertext))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(plaintext, ciphertext)

	return unpad(plaintext)
}

// AES CBC 암호화
func EncryptAES(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	iv := make([]byte, aes.BlockSize)
	_, err = rand.Read(iv)
	if err != nil {
		return nil, err
	}

	plaintext = pad(plaintext)
	ciphertext := make([]byte, len(plaintext))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, plaintext)

	return append(iv, ciphertext...), nil
}

// PKCS7 padding
func pad(src []byte) []byte {
	padLen := aes.BlockSize - len(src)%aes.BlockSize
	return append(src, bytes.Repeat([]byte{byte(padLen)}, padLen)...)
}

// unpadding
func unpad(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, fmt.Errorf("empty input")
	}
	padLen := int(src[len(src)-1])
	if padLen > aes.BlockSize || padLen > len(src) {
		return nil, fmt.Errorf("invalid padding")
	}
	return src[:len(src)-padLen], nil
}
