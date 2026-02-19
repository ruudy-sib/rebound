package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

var (
	successCount int64
	failCount    int64
)

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	var payload map[string]interface{}
	json.Unmarshal(body, &payload)

	count := atomic.AddInt64(&successCount, 1)
	fmt.Printf("[%s] ✓ /webhook (#%d) customer=%s event=%v\n",
		time.Now().Format("15:04:05"),
		count,
		r.Header.Get("X-Webhook-Secret"),
		payload["event_type"],
	)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func failHandler(w http.ResponseWriter, r *http.Request) {
	io.ReadAll(r.Body)
	r.Body.Close()

	count := atomic.AddInt64(&failCount, 1)
	fmt.Printf("[%s] ✗ /fail  (#%d) returning 500\n",
		time.Now().Format("15:04:05"),
		count,
	)

	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(`{"status":"error"}`))
}

func main() {
	http.HandleFunc("/webhook", webhookHandler)
	http.HandleFunc("/fail", failHandler)

	fmt.Println("=== Webhook Receiver listening on :8090 ===")
	fmt.Println("  GET /webhook → 200 OK")
	fmt.Println("  GET /fail    → 500 Error")
	fmt.Println()

	log.Fatal(http.ListenAndServe(":8090", nil))
}
