package main

import (
	"fmt"
	"log"
	"net/http"
)

const agentAddr = "127.0.0.1:17777"

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Processing health request")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		if _, err := fmt.Fprintln(w, "ok"); err != nil {
			log.Printf("Failed to write health response: %v", err)
		}
	})

	log.Printf("Agent listening on http://%s", agentAddr)
	if err := http.ListenAndServe(agentAddr, mux); err != nil {
		log.Fatal(err)
	}
}
