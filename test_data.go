package main

import (
	"log"

	"golang.org/x/crypto/ed25519"
)

var publicKey ed25519.PublicKey
var privateKey ed25519.PrivateKey

var testDevice device
var testUsers map[string]user

func init() {
	var err error
	publicKey, privateKey, err = ed25519.GenerateKey(nil)
	if err != nil {
		log.Fatal(err)
	}
	testDevice = device{
		id:  "foobar",
		key: publicKey,
	}
	testUsers = map[string]user{testDevice.id: user{id: "fooUser", device: testDevice}}
}
