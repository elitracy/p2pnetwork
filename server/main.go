package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "meshnet"
	keyringUser    = "server_peerskey"
	envKeyName     = "MESHNET_SERVER_PEERS_KEY"
	peersFile      = "server_peers.json.enc"
)

type Device struct {
	Name     string `json:"name"`
	PubKey   string `json:"pub_key"`
	Endpoint string `json:"endpoint"`
	IP       string `json:"ip"`
	LastSeen string `json:"last_seen"`
}

type RegisterRequest struct {
	Name      string `json:"name"`
	PubKey    string `json:"pub_key"`
	Endpoint  string `json:"endpoint"`
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
}

var (
	devices   = make(map[string]Device) // map by pubkey
	devicesMu sync.Mutex
	aesKey    []byte
)

func RegisterDevice(name, pubKey, endpoint, ip string) (*Device, error) {
	device := Device{
		Name:     name,
		PubKey:   pubKey,
		Endpoint: endpoint,
		IP:       ip,
		LastSeen: time.Now().UTC(),
	}
	result := db.Create(&device)
	return &device, result.Error
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// TODO: Verify signature and timestamp here (omitted for brevity)

	devicesMu.Lock()
	defer devicesMu.Unlock()

	newDevice := Device{
		Name:     req.Name,
		PubKey:   req.PubKey,
		Endpoint: req.Endpoint,
		IP:       r.RemoteAddr,
		LastSeen: time.Now().UTC().Format(time.RFC3339),
	}

	if _, ok := devices[req.PubKey]; ok && !compareDevices(devices[req.PubKey], newDevice) {
		http.Error(w, "Error: There is an imposter among us!!", http.StatusBadRequest)
		return
	}

	devices[req.PubKey] = newDevice

	if err := saveDevices(); err != nil {
		http.Error(w, "failed to save devices: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "registered")
}

func peersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	devicesMu.Lock()
	defer devicesMu.Unlock()

	var peerList []Device
	for _, d := range devices {
		peerList = append(peerList, d)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(peerList)
}

func saveDevices() error {
	plaintext, err := json.Marshal(devices)
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return os.WriteFile(peersFile, ciphertext, 0600)
}

func loadDevices() error {
	data, err := os.ReadFile(peersFile)
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	if len(data) < gcm.NonceSize() {
		return fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return err
	}

	var loadedDevices map[string]Device
	if err := json.Unmarshal(plaintext, &loadedDevices); err != nil {
		return err
	}

	devices = loadedDevices
	return nil
}

func compareDevices(a Device, b Device) bool {
	if a.PubKey != b.PubKey {
		return false
	}

	if a.Endpoint != b.Endpoint {
		return false
	}

	if a.IP != b.IP {
		return false
	}

	if a.Name != b.Name {
		return false
	}

	return true
}

func getOrCreateAESKey() ([]byte, error) {
	// Try keyring first
	keyB64, err := keyring.Get(keyringService, keyringUser)
	if err == nil && keyB64 != "" {
		key, err := base64.StdEncoding.DecodeString(keyB64)
		if err == nil && len(key) == 32 {
			return key, nil
		}
		log.Println("Invalid key format in keyring, regenerating")
	}

	// Try environment variable fallback
	keyB64 = os.Getenv(envKeyName)
	if keyB64 != "" {
		key, err := base64.StdEncoding.DecodeString(keyB64)
		if err == nil && len(key) == 32 {
			log.Println("Using AES key from environment variable")
			return key, nil
		}
		log.Println("Invalid key format in environment variable")
	}

	// Generate new key
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	keyB64 = base64.StdEncoding.EncodeToString(key)

	// Try saving to keyring
	if err := keyring.Set(keyringService, keyringUser, keyB64); err == nil {
		log.Println("Generated new AES key and stored in OS keyring")
	} else {
		log.Printf("Could not store AES key in keyring: %v", err)
		log.Printf("Please set environment variable %s with this key:", envKeyName)
		fmt.Println(keyB64)
	}

	return key, nil
}

func main() {
	var err error
	aesKey, err = getOrCreateAESKey()
	if err != nil {
		log.Fatalf("Failed to load AES key: %v", err)
	}

	// Load existing devices from disk
	if err := loadDevices(); err != nil {
		log.Printf("Warning: failed to load devices from disk: %v", err)
	}

	go http.HandleFunc("/register", registerHandler)
	go http.HandleFunc("/peers", peersHandler)

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
