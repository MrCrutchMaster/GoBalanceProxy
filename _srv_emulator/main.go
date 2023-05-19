package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

func baseHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("baseHandler", r.Host, r.Method, r.RequestURI)
	io.WriteString(w, "Server emulator: OK\n")
}

func startServer(port int) {
	addr := fmt.Sprintf(":%d", port)
	err := http.ListenAndServe(addr, nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}
func main() {
	http.HandleFunc("/", baseHandler)
	c := make(chan int)
	go startServer(3001)
	go startServer(3002)
	_ = <-c
}
