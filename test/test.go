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

// CorsMiddleware wraps an http.Handler to inject CORS headers and handle preflight lookups
func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow any frontend origin to read this data
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Define the permitted HTTP methods
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		// Define the permitted custom request headers (like content-type or auth tags)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		// Crucial: Instantly intercept and handle the browser's preflight OPTIONS handshake
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Pass the validated request down to your dummyHandler
		next.ServeHTTP(w, r)
	})
}

func main() {
	// Create a new ServeMux router to cleanly hold your routes
	mux := http.NewServeMux()

	// Catch all routes on this dummy server mapping
	mux.HandleFunc("/", dummyHandler)

	port := ":8080"
	log.Printf("Dummy Target Server is listening on port %s...", port)

	// Wrap your entire mux router inside the CorsMiddleware function wrapper
	if err := http.ListenAndServe(port, CorsMiddleware(mux)); err != nil {
		log.Fatalf("Dummy server failed: %v", err)
	}
}
