package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Define a private custom type for context keys to avoid any framework collisions
type contextKey string

const userIPKey contextKey = "userIP"

var codes sync.Map

type IpEntry struct {
	URL     *url.URL
	ExiryAt time.Time
}

// A thread-safe map to store IP -> Target *url.URL mappings
var ipRegistry sync.Map

func generateUniqueNumber(lengthOfCode int) string {
	str := ""
	for {
		num := rand.Intn(10) // Change 100 to your desired range
		str += strconv.Itoa(num)
		if len(str) == lengthOfCode {
			if _, exists := codes.Load(str); exists {
				str = ""
				continue
			}
			return str
		}
	}
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
			userUniqueCode := r.Context().Value(userIPKey)
			userCode, ok := userUniqueCode.(string)
			if !ok {
				fmt.Println("failed to parse ip address got from ctx into string format")
				r.URL.Scheme = "" // Forces proxy tracking execution blocks to halt
				return
			}

			// Cross reference validated identifier configuration settings
			if val, exists := ipRegistry.Load(userCode); exists {
				deviceEntry, ok := val.(IpEntry)
				if ok {
					r.URL.Scheme = deviceEntry.URL.Scheme
					r.URL.Host = deviceEntry.URL.Host
					r.Host = deviceEntry.URL.Host

				}
			} else {
				// CRITICAL PROTECTION LAYER: Halts forwarding if state validation matches are blank
				r.URL.Scheme = ""
			}
		},
		// CRITICAL FIX: Strip away browser caching properties on proxied routes
		ModifyResponse: func(res *http.Response) error {
			// Force the client browser to validate with the server on every click
			res.Header.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
			res.Header.Set("Pragma", "no-cache")
			res.Header.Set("Expires", "0")
			return nil
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

		ipRegistry.Clear()

		w.Write([]byte("devices history cleared successfully"))
	})

	// Unified API Router Endpoint Entry Point
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Bypass processing if root endpoint is hitting system monitoring routes
		if r.URL.Path == "/devices" || r.URL.Path == "/purgebuckets" {
			return
		}

		// Extract isolated client tracker layout using clean string operations
		targetUrlParam := r.URL.Query().Get("url")
		userUniqueCode := r.URL.Query().Get("code")

		if targetUrlParam == "" && userUniqueCode == "" {
			w.Write([]byte("either user code or registration url must be supplied"))
			return
		}

		// 1. Dynamic User Device Registration Logic Path
		if targetUrlParam != "" {
			formattedTarget := ensureScheme(targetUrlParam)
			//only allow localhost to be used for preventing unwanted bots
			if !strings.Contains(formattedTarget, "localhost") {
				w.Write([]byte("only localhost paths allowed"))
				return
			}
			userParsedIpAddr, err := url.Parse(formattedTarget)
			if err != nil || userParsedIpAddr.Host == "" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Invalid target routing URL layout parameter string match structure"))
				return
			}

			// Map true isolated desktop tracking IP key to target destination url configuration values
			// //now generate some unique code otp to map each device uniquely
			otp := generateUniqueNumber(6)
			ipRegistry.Store(otp, IpEntry{
				URL:     userParsedIpAddr,
				ExiryAt: time.Now().Add(24 * time.Hour * 7),
			})

			fmt.Printf("Registered isolated IP %s -> %s\n", otp, userParsedIpAddr.String())
			w.Write([]byte(fmt.Sprintf("device registered successfully with otp : %s -> %s", otp, userParsedIpAddr)))
			return
		}

		// 2. State Validation Security Logic Gate
		_, targetRegistered := ipRegistry.Load(userUniqueCode)
		if !targetRegistered {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, "device not registered yet")
			return
		}

		// 3. Execution Context Propagation & Downstream Pipeline Connection
		updatedCtx := context.WithValue(r.Context(), userIPKey, userUniqueCode)
		proxy.ServeHTTP(w, r.WithContext(updatedCtx))
	})

	// Retrieve active connected inventory maps
	mux.HandleFunc("/devices", func(w http.ResponseWriter, r *http.Request) {
		ipsConnected := map[string]string{}

		ipRegistry.Range(func(keyAny, valueAny any) bool {
			key, ok := keyAny.(string)
			if !ok {
				w.Write([]byte("failed to get devices info"))
			}
			value, ok := valueAny.(IpEntry)
			if !ok {
				w.Write([]byte("failed to get devices info,failed to convert value of devices into url formay"))

			}
			ipsConnected[key] = value.URL.String()

			return true
		})

		w.Header().Set("Content-Type", "application/json")
		resp, err := json.Marshal(map[string]any{
			"Message": "Saved Ips",
			"IPS":     ipsConnected,
		})

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
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
