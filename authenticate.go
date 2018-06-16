package main

import "errors"

type authenticator struct {
	tokens      map[string]struct{}
	subscribers map[string]chan struct{}
	store       store
}

func newAuth() authenticator {
	return authenticator{
		tokens:      make(map[string]struct{}),
		subscribers: make(map[string]chan struct{}, 0),
		store:       newStore(),
	}
}

func (a authenticator) registerToken(token string) {
	a.tokens[token] = struct{}{}
}

func (a authenticator) subscribe(token string) chan struct{} {
	ch := make(chan struct{})
	a.subscribers[token] = ch
	return ch
}

var errNoSuchToken = errors.New("Token not found")

func (a authenticator) notify(token string) error {
	ch, found := a.subscribers[token]
	if !found {
		return errNoSuchToken
	}
	ch <- struct{}{}
	return nil
}

func (a authenticator) isTokenValid(token string) bool {
	_, found := a.tokens[token]
	return found
}

func (a authenticator) remove(token string) {
	delete(a.tokens, token)
	delete(a.subscribers, token)
}

func (a authenticator) validateUser(req authRequest) error {
	return a.store.validateUser(req)
}
