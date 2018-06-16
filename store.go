package main

import (
	"errors"

	"encoding/base64"

	"golang.org/x/crypto/ed25519"
)

type user struct {
	id     string
	device device
}

type device struct {
	id   string
	name string
	key  []byte
}

type store struct {
	users map[string]user // device id to user
}

func newStore() store {
	return store{users: testUsers}
}

var errNoSuchUser = errors.New("User not found")
var errInvalidSignature = errors.New("Signature is not valid")

func (s store) validateUser(req authRequest) error {
	user, ok := s.users[req.DeviceID]
	if !ok {
		return errNoSuchUser
	}
	dec, err := base64.StdEncoding.DecodeString(req.Signature)
	if err != nil {
		return err
	}
	if !ed25519.Verify(user.device.key, []byte(req.Message), []byte(dec)) {
		return errInvalidSignature
	}
	return nil
}
