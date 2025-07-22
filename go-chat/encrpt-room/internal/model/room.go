package model

import "crypto/ecdsa"

type Room struct {
	Id      string
	PubKey  *ecdsa.PublicKey
	PriKey  *ecdsa.PrivateKey
	Clients map[string]ClientInterface
}
