package main

import (
	"log"
	"net/http"
	"strings"
	"time"
)

func requestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		pubkey_b64 := strings.Split(authHeader, " ")[1]

		device, err := GetDeviceByPubKey(pubkey_b64)
		if err != nil {
			http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
			return
		}

		if device != nil {
			// update connection
			device.LastSeen = time.Now().UTC()
			UpsertDevice(*device)
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	var err error

	err = generateServerKeys()
	if err != nil {
		log.Fatal("Error generating keys:", err.Error())
	}

	mux := http.NewServeMux()
	middlewareMux := requestMiddleware(mux)

	err = initDB()
	if err != nil {
		log.Fatal("DB Error:", err.Error())
	}

	go mux.HandleFunc("/register", registrationHandler)
	go mux.HandleFunc("/peers", peersHandler)

	log.Println("🌒Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", middlewareMux))

}
