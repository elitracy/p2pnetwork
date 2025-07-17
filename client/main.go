package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/elitracy/p2pnetwork/shared"
	"github.com/zalando/go-keyring"
)

const (
	keyringService = "meshnet"
	keyringUser    = "peerskey"
	envKeyName     = "MESHNET_PEERS_KEY"
)

const peersFile = "peers.json.enc"
const peersKeyFile = "peerskey.txt" // 32 bytes base64 AES key

type AuthInfo struct {
	public_key  ed25519.PublicKey
	private_key ed25519.PrivateKey
}

var BearerKeys AuthInfo
var peers []models.Device

func getPeersFromServer(server string) {
	for {
		req, err := http.NewRequest("GET", server+"/peers", nil)

		pubKeyStr := base64.StdEncoding.EncodeToString(BearerKeys.public_key)
		req.Header.Add("Authorization", "Bearer "+pubKeyStr)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("❌ Failed to fetch peers: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		err = json.NewDecoder(resp.Body).Decode(&peers)

		resp.Body.Close()
		if err != nil {
			log.Printf("❌ Failed to parse peer list: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		saveEncryptedPeers(peers)
		time.Sleep(2 * time.Second)
	}
}

func saveEncryptedPeers(peers []models.Device) {
	key := getOrCreateAESKey()
	plaintext, err := json.Marshal(peers)
	if err != nil {
		log.Printf("❌ Failed to marshal peers: %v", err)
		return
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		log.Printf("❌ Failed to create cipher: %v", err)
		return
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Printf("❌ Failed to create GCM: %v", err)
		return
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		log.Printf("❌ Failed to generate nonce: %v", err)
		return
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	if err := os.WriteFile(peersFile, ciphertext, 0600); err != nil {
		log.Printf("❌ Failed to write encrypted peer file: %v", err)
		return
	}
}

func getOrCreateAESKey() []byte {
	// 1) Try keyring first
	keyB64, err := keyring.Get(keyringService, keyringUser)
	if err == nil && keyB64 != "" {
		key, err := base64.StdEncoding.DecodeString(keyB64)
		if err == nil && len(key) == 32 {
			return key
		}
		log.Printf("⚠️ Invalid key format in keyring, regenerating")
	}

	// 2) Try environment variable fallback
	keyB64 = os.Getenv(envKeyName)
	if keyB64 != "" {
		key, err := base64.StdEncoding.DecodeString(keyB64)
		if err == nil && len(key) == 32 {
			log.Printf("⚠️ Using AES key from environment variable %s", envKeyName)
			return key
		}
		log.Printf("⚠️ Invalid key format in environment variable %s", envKeyName)
	}

	// 3) Generate new key
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		log.Fatalf("❌ Failed to generate AES key: %v", err)
	}
	keyB64 = base64.StdEncoding.EncodeToString(key)

	// Try saving to keyring
	if err := keyring.Set(keyringService, keyringUser, keyB64); err == nil {
		log.Printf("🔑 Generated new AES key and stored in OS keyring")
	} else {
		log.Printf("⚠️ Could not store AES key in keyring: %v", err)
		log.Printf("⚠️ Please set environment variable %s with this key:", envKeyName)
		fmt.Println(keyB64)
	}

	return key
}

func getDeviceByPubKey(pubKey string) (*models.Device, error) {
	for _, peer := range peers {
		if peer.PubKey == pubKey {
			fmt.Println("trying to return peer")
			return &peer, nil
		}
	}

	return nil, errors.New("Could not find peer")
}

func requestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Grab Authorization header (or any other)
		authHeader := r.Header.Get("Authorization")
		pubkey_b64 := strings.Split(authHeader, " ")[1]

		// check if exists
		device, err := getDeviceByPubKey(pubkey_b64)
		if err != nil {
			http.Error(w, "failed to get device: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if device != nil {
			device.LastSeen = time.Now().UTC()
			device.Connected = true // doesn't tell server
		}

		// Pass the request to the next handler
		next.ServeHTTP(w, r)
	})
}

func checkPeers() {
	for {
		for i, peer := range peers {

			req, err := http.NewRequest("GET", peer.Endpoint+"/ping", nil)

			pubKeyStr := base64.StdEncoding.EncodeToString(BearerKeys.public_key)
			req.Header.Add("Authorization", "Bearer "+pubKeyStr)
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)

			if err != nil || resp.StatusCode != 200 {
				log.Printf("❌ Failed to ping peer: %v", err)
				peers[i].Connected = false
			} else {
				peers[i].Connected = true
			}
		}

		fmt.Println("🔄 Connected Peers:")
		for _, peer := range peers {
			if peer.Connected {
				fmt.Printf("- %s @ %s (%s:%s)\n", peer.Name, peer.IP, peer.Endpoint, peer.Port)
			}
		}

		time.Sleep(time.Second * 5)
	}
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
}

func main() {
	if len(os.Args) != 5 {
		log.Fatal("Usage: ./client <device_name> <host> <port> <control_server_url>")
	}
	name := os.Args[1]
	host := os.Args[2]
	port := os.Args[3]
	server := os.Args[4]

	err := registerDevice(name, host+":"+port, server, &BearerKeys)
	if err != nil {
		log.Fatalf("Registration failed: %v", err)
	}

	go getPeersFromServer(server)
	go checkPeers()

	mux := http.NewServeMux()
	middlewareMux := requestMiddleware(mux)

	go mux.HandleFunc("/ping", handlePing)

	log.Printf("Server running on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, middlewareMux))

}
