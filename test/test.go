package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

// Response structure to send back to the client
type ApiResponse struct {
	Message string              `json:"message"`
	Method  string              `json:"received_method"`
	Headers map[string][]string `json:"received_headers"`
	Body    string              `json:"received_body"`
}

func dummyHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("--> Dummy Server received a %s request on path: %s", r.Method, r.URL.Path)

	// Read the incoming request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Prepare a JSON response showing everything we received
	responseData := ApiResponse{
		Message: "Success! The proxy successfully delivered your request.",
		Method:  r.Method,
		Headers: r.Header,
		Body:    string(bodyBytes),
	}

	// Send back JSON headers and payload
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(responseData)
}

func main() {
	// Catch all routes on this dummy server
	http.HandleFunc("/", dummyHandler)

	port := ":8080"
	log.Printf("Dummy Target Server is listening on port %s...", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Dummy server failed: %v", err)
	}
}
