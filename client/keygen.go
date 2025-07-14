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

func ensureKeysExist(pubPath string, privPath string) (ed25519.PublicKey, ed25519.PrivateKey, error) {
	// Check if both files exist
	pubExists := fileExists(pubPath)
	privExists := fileExists(privPath)

	if pubExists && privExists {
		return loadKeys(pubPath, privPath)
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

	fmt.Println("🔑 Generated new keypair")
	return pub, priv, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func loadKeys(pubkeyPath string, privkeyPath string) (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pubKeyB64, err := os.ReadFile(pubkeyPath)
	if err != nil {
		return nil, nil, err
	}
	privKeyB64, err := os.ReadFile(privkeyPath)
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

func registerDevice(name string, endpoint string, server string, authInfo *AuthInfo) error {
	pubKeyPath := "pubkey-" + name + ".txt"
	privKeyPath := "privkey-" + name + ".txt"
	pubKey, privKey, err := ensureKeysExist(pubKeyPath, privKeyPath)

	authInfo.public_key = pubKey
	authInfo.private_key = privKey

	if err != nil {
		return fmt.Errorf("failed to load or generate keys: %v", err)
	}

	timestamp := time.Now().Unix()
	msg := []byte(strconv.FormatInt(timestamp, 10))
	sig := ed25519.Sign(privKey, msg)

	registerReq := RegisterRequest{
		Name:      name,
		PubKey:    base64.StdEncoding.EncodeToString(pubKey),
		Endpoint:  endpoint,
		Timestamp: timestamp,
		Signature: base64.StdEncoding.EncodeToString(sig),
	}
	body, err := json.Marshal(registerReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", server+"/device", bytes.NewReader(body))
	if err != nil {
		return err
	}
	pubKeyStr := base64.StdEncoding.EncodeToString(authInfo.public_key)

	req.Header.Add("Authorization", "Bearer "+pubKeyStr)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error on response.\n[ERROR] -", err)
	}

	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server error: %s", string(respBody))
	}

	fmt.Println("✅ Device registered successfully.")
	return nil
}
