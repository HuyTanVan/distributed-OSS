package main

import (
	"io"
	"log"
	"net/http"
	"sync"
)

var (
	nodes = []string{
		// "http://localhost:8081",
		// "http://localhost:8082",
		// "http://localhost:8083",
	}
	nodes        = []string{}
	currentIndex = 0
	mu           sync.Mutex
)

// a simple load balancer with round-robin algorithm
func main() {
	http.HandleFunc("/", handleRequest)
	log.Println("load balancer started on :8080")

	for i, node := range nodes {
		log.Printf("backend node %d: %s", i+1, node)
	}
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	// Pick a node (round-robin)
	node := pickNode()

	// Create proxy request
	targetURL := node + r.URL.Path
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	log.Printf("[%s] %s %s -> %s", r.RemoteAddr, r.Method, r.URL.Path, node)

	proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, "Proxy error: "+err.Error(), 500)
		return
	}

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Send request to backend
	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		log.Printf("Backend error: %v", err)
		http.Error(w, "Backend error: "+err.Error(), 502)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Copy status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	io.Copy(w, resp.Body)
}

func pickNode() string {
	mu.Lock()
	defer mu.Unlock()

	node := nodes[currentIndex]
	currentIndex = (currentIndex + 1) % len(nodes)
	return node
}
