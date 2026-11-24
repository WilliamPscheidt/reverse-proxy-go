package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

type BackendResponse struct {
	Server    string            `json:"server"`
	Status    string            `json:"status"`
	Path      string            `json:"path,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	Headers   map[string]string `json:"headers,omitempty"`
}

func main() {
	httpPort := flag.Int("http-port", 8081, "HTTP port to listen on")
	tcpPort := flag.Int("tcp-port", 9081, "TCP port to listen on")
	name := flag.String("name", "backend-1", "Backend server identifier")
	delay := flag.Duration("delay", 0, "Artificial response delay")
	flag.Parse()

	var healthy atomic.Bool
	healthy.Store(true)

	// HTTP Server
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if !healthy.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("UNHEALTHY"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	mux.HandleFunc("/toggle-health", func(w http.ResponseWriter, r *http.Request) {
		newVal := !healthy.Load()
		healthy.Store(newVal)
		state := "HEALTHY"
		if !newVal {
			state = "UNHEALTHY"
		}
		log.Printf("[%s] Health manually toggled to: %s", *name, state)
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "State toggled to %s\n", state)
	})

	fastPayload := []byte(fmt.Sprintf(`{"status":"ok","server":"%s"}`+"\n", *name))
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fastPayload)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if *delay > 0 {
			time.Sleep(*delay)
		}

		headers := make(map[string]string)
		for k, v := range r.Header {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}

		resp := BackendResponse{
			Server:    *name,
			Status:    "success",
			Path:      r.URL.Path,
			Timestamp: time.Now(),
			Headers:   headers,
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Backend-Server", *name)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})

	httpAddr := fmt.Sprintf(":%d", *httpPort)
	go func() {
		log.Printf("[%s] HTTP Backend listening on %s", *name, httpAddr)
		if err := http.ListenAndServe(httpAddr, mux); err != nil {
			log.Printf("[%s] HTTP server stopped: %v", *name, err)
		}
	}()

	// TCP Echo Server
	tcpAddr := fmt.Sprintf(":%d", *tcpPort)
	tcpLn, err := net.Listen("tcp", tcpAddr)
	if err == nil {
		go func() {
			log.Printf("[%s] TCP Backend listening on %s", *name, tcpAddr)
			for {
				conn, err := tcpLn.Accept()
				if err != nil {
					return
				}
				go func(c net.Conn) {
					defer c.Close()
					_, _ = io.Copy(c, c)
				}(conn)
			}
		}()
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	if tcpLn != nil {
		_ = tcpLn.Close()
	}
	log.Printf("[%s] Shutting down backend", *name)
}
