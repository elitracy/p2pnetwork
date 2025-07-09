package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type RegisterRequest struct {
	Name      string `json:"name"`
	PubKey    string `json:"pub_key"`
	Endpoint  string `json:"endpoint"`
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
}

func ensureKeysExist() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pubPath := "pubkey.txt"
	privPath := "privkey.txt"

	// Check if both files exist
	pubExists := fileExists(pubPath)
	privExists := fileExists(privPath)

	if pubExists && privExists {
		return loadKeys()
	}

	// Generate new keypair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	// Encode and write to files
	pubB64 := base64.StdEncoding.EncodeToString(pub)
	privB64 := base64.StdEncoding.EncodeToString(priv)

	if err := os.WriteFile(pubPath, []byte(pubB64), 0600); err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(privPath, []byte(privB64), 0600); err != nil {
		return nil, nil, err
	}

	fmt.Println("🔑 Generated new keypair and saved to pubkey.txt / privkey.txt")
	return pub, priv, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func loadKeys() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pubKeyB64, err := os.ReadFile("pubkey.txt")
	if err != nil {
		return nil, nil, err
	}
	privKeyB64, err := os.ReadFile("privkey.txt")
	if err != nil {
		return nil, nil, err
	}

	pubKey, err := base64.StdEncoding.DecodeString(string(bytes.TrimSpace(pubKeyB64)))
	if err != nil {
		return nil, nil, err
	}
	privKey, err := base64.StdEncoding.DecodeString(string(bytes.TrimSpace(privKeyB64)))
	if err != nil {
		return nil, nil, err
	}

	return ed25519.PublicKey(pubKey), ed25519.PrivateKey(privKey), nil
}

func registerDevice(name, endpoint, server string) error {
	pubKey, privKey, err := ensureKeysExist()
	if err != nil {
		return fmt.Errorf("failed to load or generate keys: %v", err)
	}

	timestamp := time.Now().Unix()
	msg := []byte(strconv.FormatInt(timestamp, 10))
	sig := ed25519.Sign(privKey, msg)

	req := RegisterRequest{
		Name:      name,
		PubKey:    base64.StdEncoding.EncodeToString(pubKey),
		Endpoint:  endpoint,
		Timestamp: timestamp,
		Signature: base64.StdEncoding.EncodeToString(sig),
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := http.Post(server+"/register", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server error: %s", string(respBody))
	}

	fmt.Println("✅ Device registered successfully.")
	return nil
}
