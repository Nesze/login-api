package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type authRequest struct {
	DeviceID  string `json:"deviceId"`
	Message   string `json:"message"`
	Signature string `json:"signature"` // base64 encoded
}

type httpHandler struct {
	auth authenticator
	qr   qrService
}

func newHTTPHandler() httpHandler {
	return httpHandler{auth: newAuth()}
}

type qrCodeResponse struct {
	QRCode string
}

func (h httpHandler) qrCode(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	qrCode, err := h.qr.generatePNG(token)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.auth.registerToken(token)

	err = json.NewEncoder(w).Encode(qrCodeResponse{QRCode: base64.StdEncoding.EncodeToString(qrCode)})
	if err != nil {
		log.Println(err)
		return
	}
}

func (h httpHandler) isAuthenticated(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if ok := h.auth.isTokenValid(token); !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	ch := h.auth.subscribe(token)

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
	h.auth.remove(token)
}

func (h httpHandler) authenticate(w http.ResponseWriter, r *http.Request) {
	var authReq authRequest
	err := json.NewDecoder(r.Body).Decode(&authReq)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := h.auth.validateUser(authReq); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if err = h.auth.notify(authReq.Message); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
