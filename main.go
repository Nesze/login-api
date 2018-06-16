package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"log"

	"github.com/gorilla/mux"
)

var serverAddr = flag.String("addr", envString("ADDR", "localhost:8080"), "server addr (default is :8080))")

func main() {
	flag.Parse()
	run(context.Background(), nil)
}

func run(ctx context.Context, ready chan<- struct{}) {
	httpQuit, stopServer := initializeHTTPServer(*serverAddr, newHTTPHandler())
	if ready != nil {
		ready <- struct{}{}
	}

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-ch:
		log.Println("Received quit signal")
	case <-httpQuit:
	case <-ctx.Done():
	}
	stopServer()
	log.Println("Done. Bye")
}

func initializeHTTPServer(addr string, h httpHandler) (chan error, func()) {
	errors := make(chan error)

	router := mux.NewRouter()
	router.HandleFunc("/api/1.0/qrCode", h.qrCode).Queries("token", "").Methods(http.MethodGet)
	router.HandleFunc("/api/1.0/isAuthenticated", h.isAuthenticated).Queries("token", "").Methods(http.MethodGet, http.MethodOptions)
	router.HandleFunc("/api/1.0/authenticate", h.authenticate).Methods(http.MethodPost)

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
