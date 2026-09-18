package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
)

// Define a private custom type for context keys to avoid any framework collisions
type contextKey string

const userIPKey contextKey = "userIP"

// A thread-safe map to store IP -> Target *url.URL mappings
var ipRegistry sync.Map

// Helper utility function to cleanly isolate the true client IP from proxy networks
func getTrueClientIP(r *http.Request) string {
	// Prioritise Cloudflare's direct value if provided on Render
	if cfIP := r.Header.Get("CF-Connecting-IP"); cfIP != "" {
		return strings.TrimSpace(cfIP)
	}

	// Fallback to splitting the standard multi-hop reverse proxy chain list
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		ips := strings.Split(forwarded, ",")
		realIP := strings.TrimSpace(ips[0]) // Isolate the first element (true desktop client)

		// Strip ports if present in the forward string token layout
		if strings.Contains(realIP, ":") {
			host, _, err := net.SplitHostPort(realIP)
			if err == nil {
				return host
			}
		}
		return realIP
	}

	// Local development fallback parsing loop
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// Helper utility function to ensure targets have valid protocol network schemes
func ensureScheme(target string) string {
	target = strings.TrimSpace(target)
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		return "http://" + target // Defaults to HTTP for raw inputs like "localhost:8000"
	}
	return target
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000" // Local system architecture fallback
	}

	proxy := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			// Pull our typed key out of request context propagation
			ipAddrAny := r.Context().Value(userIPKey)
			ipAddress, ok := ipAddrAny.(string)
			if !ok {
				fmt.Println("failed to parse ip address got from ctx into string format")
				r.URL.Scheme = "" // Forces proxy tracking execution blocks to halt
				return
			}

			// Cross reference validated identifier configuration settings
			if val, exists := ipRegistry.Load(ipAddress); exists {
				redirectUrl, ok := val.(*url.URL)
				if ok {
					r.URL.Scheme = redirectUrl.Scheme
					r.URL.Host = redirectUrl.Host
					r.Host = redirectUrl.Host
				}
			} else {
				// CRITICAL PROTECTION LAYER: Halts forwarding if state validation matches are blank
				r.URL.Scheme = ""
			}
		},
	}

	mux := http.NewServeMux()

	// Clear out connected memory allocations
	mux.HandleFunc("/purgebuckets", func(w http.ResponseWriter, r *http.Request) {
		pass := r.URL.Query().Get("password")
		if pass == "" || pass != "aishafarooq" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("wrong or empty password"))
			return
		}

		// Multi-core clearing method targeting our concurrent map storage
		ipRegistry.Range(func(key, value any) bool {
			ipRegistry.Delete(key)
			return true
		})

		w.Write([]byte("devices history cleared successfully"))
	})

	// Unified API Router Endpoint Entry Point
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Bypass processing if root endpoint is hitting system monitoring routes
		if r.URL.Path == "/devices" || r.URL.Path == "/purgebuckets" {
			return
		}

		// Extract isolated client tracker layout using clean string operations
		ipAddr := getTrueClientIP(r)
		targetUrlParam := r.URL.Query().Get("url")

		// 1. Dynamic User Device Registration Logic Path
		if targetUrlParam != "" {
			formattedTarget := ensureScheme(targetUrlParam)
			userParsedIpAddr, err := url.Parse(formattedTarget)
			if err != nil || userParsedIpAddr.Host == "" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Invalid target routing URL layout parameter string match structure"))
				return
			}

			// Map true isolated desktop tracking IP key to target destination url configuration values
			ipRegistry.Store(ipAddr, userParsedIpAddr)

			fmt.Printf("Registered isolated IP %s -> %s\n", ipAddr, userParsedIpAddr.String())
			w.Write([]byte(fmt.Sprintf("device registered successfully for IP: %s", ipAddr)))
			return
		}

		// 2. State Validation Security Logic Gate
		_, targetRegistered := ipRegistry.Load(ipAddr)
		if !targetRegistered {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(fmt.Sprintf("you have not registered first. Your isolated tracker device tag is: %s", ipAddr)))
			return
		}

		// 3. Execution Context Propagation & Downstream Pipeline Connection
		updatedCtx := context.WithValue(r.Context(), userIPKey, ipAddr)
		proxy.ServeHTTP(w, r.WithContext(updatedCtx))
	})

	// Retrieve active connected inventory maps
	mux.HandleFunc("/devices", func(w http.ResponseWriter, r *http.Request) {
		ipsConnected := map[any]any{}

		ipRegistry.Range(func(key, value any) bool {

			ipsConnected[key] = value

			return true
		})

		w.Header().Set("Content-Type", "application/json")
		resp, err := json.Marshal(map[string]any{
			"Message": "Saved Ips",
			"IPS":     ipsConnected,
		})

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("failed to compile engine connected client arrays"))
			return
		}

		w.Write(resp)
	})

	fmt.Printf("Server started successfully on port %s\n", port)
	if err := http.ListenAndServe("0.0.0.0:"+port, CorsMiddleware(mux)); err != nil {
		fmt.Printf("failed to start server err is : %v\n", err)
	}
}

// Standard CORS configuration security wrapper
func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
