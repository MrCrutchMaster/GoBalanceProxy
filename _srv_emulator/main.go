package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

func baseHandler1(w http.ResponseWriter, r *http.Request) {
	fmt.Println("baseHandler", r.Host, r.Method, r.RequestURI)
	w.Header().Add("My-Header", "my value")
	w.Header().Add("My-Header", "my value 2")
	io.WriteString(w, "Server emulator 1: OK\n")
}

func baseHandler2(w http.ResponseWriter, r *http.Request) {
	fmt.Println("baseHandler", r.Host, r.Method, r.RequestURI)
	w.Header().Add("My-Header", "my value")
	w.Header().Add("My-Header", "my value 2")
	io.WriteString(w, "Server emulator 2: OK\n")
}

func baseHandler3(w http.ResponseWriter, r *http.Request) {
	fmt.Println("baseHandler", r.Host, r.Method, r.RequestURI)
	w.WriteHeader(500)
	w.Header().Add("My-Header", "my value")
	w.Header().Add("My-Header", "my value 2")
	io.WriteString(w, "Server emulator 3: NOT OK\n")
}

func startServer1(port int) {
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(baseHandler1))
	addr := fmt.Sprintf(":%d", port)
	err := http.ListenAndServe(addr, mux)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}

func startServer2(port int) {
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(baseHandler2))
	addr := fmt.Sprintf(":%d", port)
	err := http.ListenAndServe(addr, mux)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}
func startServer3(port int) {
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(baseHandler3))
	addr := fmt.Sprintf(":%d", port)
	err := http.ListenAndServe(addr, mux)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}

func main() {
	//http.HandleFunc("/", baseHandler)
	c := make(chan int)
	go startServer1(3001)
	go startServer2(3002)
	go startServer3(3003)
	_ = <-c
}
