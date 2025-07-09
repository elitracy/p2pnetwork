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
	"time"

	"github.com/zalando/go-keyring"
)

type Device struct {
	Name     string `json:"name"`
	PubKey   string `json:"pub_key"`
	Endpoint string `json:"endpoint"`
	IP       string `json:"ip"`
	LastSeen string `json:"last_seen"`
}

const (
	keyringService = "meshnet"
	keyringUser    = "peerskey"
	envKeyName     = "MESHNET_PEERS_KEY"
)

const peersFile = "peers.json.enc"
const peersKeyFile = "peerskey.txt" // 32 bytes base64 AES key

func syncPeers(server string) {
	for {
		resp, err := http.Get(server + "/peers")
		if err != nil {
			log.Printf("❌ Failed to fetch peers: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}
		var peers []Device
		err = json.NewDecoder(resp.Body).Decode(&peers)
		resp.Body.Close()
		if err != nil {
			log.Printf("❌ Failed to parse peer list: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}
		fmt.Println("🔄 Current Peers:")
		for _, peer := range peers {
			fmt.Printf("- %s (%s) @ %s\n", peer.Name, peer.IP, peer.Endpoint)
		}

		saveEncryptedPeers(peers)
		time.Sleep(2 * time.Second)
	}
}

func saveEncryptedPeers(peers []Device) {
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

func main() {
	if len(os.Args) != 4 {
		log.Fatal("Usage: ./client <device_name> <public_ip:port> <control_server_url>")
	}
	name := os.Args[1]
	endpoint := os.Args[2]
	server := os.Args[3]

	err := registerDevice(name, endpoint, server)
	if err != nil {
		log.Fatalf("Registration failed: %v", err)
	}

	syncPeers(server)
}

