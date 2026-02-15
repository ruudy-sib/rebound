package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

func main() {
	port := ":8090"

	http.HandleFunc("/webhook", webhookHandler)
	http.HandleFunc("/dlq", dlqHandler)
	http.HandleFunc("/fail", failHandler)

	log.Printf("🎧 Webhook receiver listening on http://localhost%s", port)
	log.Printf("📡 Endpoints:")
	log.Printf("   - POST /webhook - Always succeeds (200 OK)")
	log.Printf("   - POST /dlq     - Dead letter queue endpoint")
	log.Printf("   - POST /fail    - Always fails (500 Internal Server Error)")
	log.Printf("\n")

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	logRequest(r, "WEBHOOK")

	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	log.Printf("📦 Payload: %s\n", string(body))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"success","message":"Webhook received"}`)
	log.Printf("✅ Responded with 200 OK\n\n")
}

func dlqHandler(w http.ResponseWriter, r *http.Request) {
	logRequest(r, "DEAD-LETTER")

	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	log.Printf("☠️  Dead letter payload: %s\n", string(body))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"dlq_received"}`)
	log.Printf("✅ Dead letter accepted\n\n")
}

func failHandler(w http.ResponseWriter, r *http.Request) {
	logRequest(r, "FAIL")

	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	log.Printf("📦 Payload: %s\n", string(body))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(w, `{"error":"Simulated failure"}`)
	log.Printf("❌ Responded with 500 Internal Server Error (will trigger retry)\n\n")
}

func logRequest(r *http.Request, label string) {
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("📨 [%s] Received request at %s", label, time.Now().Format(time.RFC3339))
	log.Printf("🔗 Method: %s", r.Method)
	log.Printf("🔗 Path: %s", r.URL.Path)
	log.Printf("🔑 X-Message-Key: %s", r.Header.Get("X-Message-Key"))
	log.Printf("👤 User-Agent: %s", r.Header.Get("User-Agent"))
}
