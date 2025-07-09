// main.go
package main

import (
	"log"
	"net/http"
	"sync"
	"time"
)

type Device struct {
	Name     string    `json:"name"`
	PubKey   string    `json:"pub_key"`
	Endpoint string    `json:"endpoint"`
	IP       string    `json:"ip"`
	LastSeen time.Time `json:"last_seen"`
}

var (
	devices = map[string]Device{}
	mu      sync.Mutex
)

func main() {
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/peers", peerListHandler)

	log.Println("Control server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
