package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"image/png"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"log"

	"github.com/boombuler/barcode/qr"
	"github.com/gorilla/mux"
)

var serverAddr = flag.String("addr", envString("ADDR", "localhost:8080"), "server addr (default is :8080))")

func main() {
	flag.Parse()
	httpQuit, stopServer := initializeHTTPServer(*serverAddr, newHTTPHandler())

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-ch:
		log.Println("Received quit signal")
	case <-httpQuit:
	}
	stopServer()
}

type httpHandler struct {
	store  store
	tokens map[string]bool
	notify map[string]chan struct{}
}

func newHTTPHandler() httpHandler {
	//set foo for testing
	return httpHandler{store: newStore(), tokens: map[string]bool{"foo": false}, notify: map[string]chan struct{}{}}
}

func (h httpHandler) qrCode(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	qrCode, err := qr.Encode(token, qr.M, qr.Auto)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var b bytes.Buffer
	if err := png.Encode(&b, qrCode); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = w.Write([]byte(fmt.Sprintf("{\"qrCode\":\"%s\"}", base64.StdEncoding.EncodeToString(b.Bytes()))))
	if err != nil {
		log.Println(err)
		return
	}
	h.tokens[token] = false
}

func (h httpHandler) loggedIn(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, ok := h.tokens[token]; !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	//register call back channel
	ch := make(chan struct{})
	h.notify[token] = ch

	w.Header().Set("Content-type", "application/json; charset=UTF-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	// send something back instantly
	_, err := w.Write([]byte("{\"login\":\"waiting\"}\n"))
	if err != nil {
		log.Println(err)
		return
	}
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	select {
	case <-ch:
		w.Write([]byte("{\"login\":\"success\"}"))
	case <-time.After(10 * time.Second):
		w.Write([]byte("{\"login\":\"timeout\"}"))
	}
	delete(h.notify, token)
	delete(h.tokens, token)
}

func (h httpHandler) login(w http.ResponseWriter, r *http.Request) {
	accID := r.URL.Query().Get("accountId")
	if accID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	secureToken := r.URL.Query().Get("secureToken")
	if secureToken == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	token, err := h.store.validateUser(loginRequest{accountID: accID, secureToken: secureToken})
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	h.notify[token] <- struct{}{}
}

type store struct {
	users map[string]user
}

func newStore() store {
	return store{}
}

type user struct {
	id      string
	devices []device
}

type device struct {
	id   string
	name string
	key  []byte
}

func (s store) validateUser(req loginRequest) (string, error) {
	//TODO
	return "", nil
}

type loginRequest struct {
	accountID   string
	secureToken string
}

func initializeHTTPServer(addr string, h httpHandler) (chan error, func()) {
	errors := make(chan error)

	router := mux.NewRouter()
	router.HandleFunc("/api/1.0/qrCode", h.qrCode).Queries("token", "").Methods(http.MethodGet)
	router.HandleFunc("/api/1.0/loggedIn", h.loggedIn).Queries("token", "").Methods(http.MethodGet)
	router.HandleFunc("/api/1.0/login", h.login).Queries("accountId", "", "secureToken", "").Methods(http.MethodPost)

	server := &http.Server{Addr: addr, Handler: router}

	go func() {
		defer close(errors)
		log.Printf("Starting HTTP server on %s", addr)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Println(err)
			errors <- err
		}
	}()

	cancelServer := func() {
		log.Println("Closing HTTP server")
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			err := server.Close()
			if err != nil {
				log.Println(err)
			}
		}
	}

	return errors, cancelServer
}

func envString(key, def string) string {
	if env := os.Getenv(key); env != "" {
		return env
	}
	return def
}
