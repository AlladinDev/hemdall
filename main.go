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
	"sync"
)

var devices = map[string]string{}

// A thread-safe map to store IP -> Target URL mappings
var ipRegistry sync.Map

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000" // Local fallback
	}
	proxy := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			//first get the ip address of the user
			ipAddrAny := r.Context().Value("userIP")
			ipAddress, ok := ipAddrAny.(string)
			if !ok {
				fmt.Println("failed to parse ip address got from ctx into string format")
			}

			//now we have to check if this ipAddr is present in ip registery or not i mean whether it is registered first or not
			if val, exists := ipRegistry.Load(ipAddress); exists {

				redirectUrl, ok := val.(*url.URL)
				if ok {
					fmt.Print(val)
					//now we have to dynamically assign values to this incoming request from the saved request
					r.URL.Scheme = redirectUrl.Scheme
					r.URL.Host = redirectUrl.Host
					r.Host = redirectUrl.Host
				}
			}

		},
	}

	defer func() {
		if err := recover(); err != nil {
			fmt.Println("panicked but recovered inside main err is : ", err)
		}
	}()

	mux := http.NewServeMux()

	mux.HandleFunc("/purgebuckets", func(w http.ResponseWriter, r *http.Request) {
		//first get the query password
		pass := r.URL.Query().Get("password")
		if pass == "" || pass != "aishafarooq" {
			w.Write([]byte("wrong or empty password"))
		}

		ipRegistry.Clear()
		w.Write([]byte("devices history cleared successfully"))
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		targetUrl := r.URL.Query().Get("url")
		ipAddr := ""
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ipAddr = forwarded
		}

		if ipAddr == "" {
			//if not this header then get the ip address from req object
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				w.Write([]byte(err.Error()))
				return
			}

			ipAddr = ip
		}
		if targetUrl != "" {
			//if port is not empty it means user wants to register endpoint
			// //now make this port into localhost url format
			userParsedIpAddr, err := url.Parse(targetUrl)
			if err != nil {
				w.Write([]byte(err.Error()))
				return
			}

			ipRegistry.Store(ipAddr, userParsedIpAddr)

			w.Write([]byte("device registered successfully"))
			return
		}

		//now here it means user wants to use our server as proxy so now find if he has registered his ip or not
		_, targetRegistered := ipRegistry.Load(ipAddr)
		if !targetRegistered {
			w.Write([]byte("you have not registerd first"))
			return
		}

		updatedCtx := context.WithValue(r.Context(), "userIP", ipAddr)

		proxy.ServeHTTP(w, r.WithContext(updatedCtx))
	})

	mux.HandleFunc("/devices", func(w http.ResponseWriter, r *http.Request) {
		ipsConnected := []any{}
		ipRegistry.Range(func(key, value any) bool {
			ipsConnected = append(ipsConnected, key)
			return true
		})

		resp, err := json.Marshal(map[string]any{
			"Message": "Saved Ips",
			"IPS":     ipsConnected,
		})

		if err != nil {
			fmt.Print(err.Error())
			w.Write([]byte("failed to get devices connected"))
			return
		}

		w.Write(resp)
	})
	fmt.Printf("Server started successfully")
	if err := http.ListenAndServe("0.0.0.0:"+port, CorsMiddleware(mux)); err != nil {
		fmt.Println("failed to start server err is : ", err)
	}
}

func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow any origin, or specify a domain like "http://localhost:3000"
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Define the permitted methods
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		// Define the permitted request headers
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		// Crucial: Handle the browser's preflight OPTIONS request immediately
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Pass the request along the middleware chain
		next.ServeHTTP(w, r)
	})
}
