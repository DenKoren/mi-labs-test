package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

const (
	bootLag     = 5 * time.Second
	healthyLag  = 15 * time.Second
	responseLag = 15 * time.Second
	listenPort  = 8080
)

var (
	isHealthy bool
)

func main() {
	time.AfterFunc(healthyLag, func() { isHealthy = true })

	mux := http.NewServeMux()

	mux.HandleFunc("/health", handleHealthcheck)
	mux.HandleFunc("/calculate/", handleCalculate)

	// Imitate long container bootup: don't listen port for several seconds.
	time.Sleep(bootLag)

	addr := fmt.Sprintf(":%d", listenPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen '%s': %v", addr, err)
	}
	log.Printf("Server listening at '%s'", addr)

	err = http.Serve(lis, mux)
	log.Printf("Server stopped listening at '%s': %v", addr, err)
}

func handleHealthcheck(w http.ResponseWriter, r *http.Request) {
	if !isHealthy {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	w.WriteHeader(http.StatusOK)
	return
}

func handleCalculate(w http.ResponseWriter, r *http.Request) {
	if !isHealthy {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	time.Sleep(responseLag)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(r.RequestURI))
	return
}
