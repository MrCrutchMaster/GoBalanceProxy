package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

func handler(i int, resp string, statusCode int) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		//time.Sleep(3000 * time.Millisecond)
		//fmt.Printf("baseHandler %d %s %s %s\n", i, r.Host, r.Method, r.RequestURI)
		w.WriteHeader(statusCode)
		w.Header().Add("My-Header", "my value")
		w.Header().Add("My-Header", "my value 2")
		s := fmt.Sprintf("Server emulator %d: %s\n", i, resp)
		io.WriteString(w, s)
	}
}

func startServer(port int, resp string, statusCode int) {
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(handler(port, resp, statusCode)))
	addr := fmt.Sprintf(":%d", port)
	err := http.ListenAndServe(addr, mux)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("proxy closed\n")
	} else if err != nil {
		fmt.Printf("error starting proxy: %s\n", err)
		os.Exit(1)
	}
}

func main() {
	c := make(chan int)
	go startServer(3001, "OK", 200)
	go startServer(3002, "OK", 200)
	go startServer(3003, "NOT OK", 500)
	go startServer(3004, "NOT OK", 301)
	go startServer(3005, "OK", 200)

	_ = <-c
}
