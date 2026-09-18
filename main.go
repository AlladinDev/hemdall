package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

var devices = map[string]string{}

// A thread-safe map to store IP -> Target URL mappings
var ipRegistry sync.Map

func main() {
	proxy := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			//first get the ip address of the user
			ipAddr, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ipAddr = r.RemoteAddr
			}

			//now maybe the user is behind load balancer so extract the real ip
			// If behind a proxy/load balancer, prioritize the real client IP [1]
			if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
				ipAddr = forwarded
			}

			//now we have to check if this ipAddr is present in ip registery or not i mean whether it is registered first or not
			if val, exists := ipRegistry.Load(ipAddr); exists {

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

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		targetUrl := r.URL.Query().Get("url")
		ipAddr, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ipAddr = r.RemoteAddr
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

		proxy.ServeHTTP(w, r)
	})

	fmt.Printf("Server started successfully")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		fmt.Println("failed to start server err is : ", err)
	}
}
