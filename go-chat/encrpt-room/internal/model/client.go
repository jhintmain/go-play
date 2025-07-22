package model

import (
	"crypto/ecdsa"
	"github.com/gorilla/websocket"
)

type ClientInterface interface {
	GetID() string
	Send(msg []byte)
	GetNickname() string
	GetConn() *websocket.Conn
	GetPubKey() *ecdsa.PublicKey
}

type Client struct {
	ID       string
	Conn     *websocket.Conn
	Nickname string
	PubKey   *ecdsa.PublicKey
	SendChan chan []byte
}

func (client *Client) GetID() string {
	return client.ID
}

func (client *Client) GetConn() *websocket.Conn {
	return client.Conn
}

func (client *Client) GetNickname() string {
	return client.Nickname
}

func (client *Client) GetPubKey() *ecdsa.PublicKey {
	return client.PubKey
}

func (client *Client) Send(msg []byte) {
	client.SendChan <- msg
}
