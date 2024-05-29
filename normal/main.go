package main

import (
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
)

func main() {
	// reuse
	pid := os.Getpid()
	l, err := net.Listen("tcp", ":18081")
	if err != nil {
		panic(err)
	}
	server := &http.Server{}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Hello from PID %d \n", pid)
	})
	fmt.Printf("HTTP Server with PID: %d is running \n", pid)
	panic(server.Serve(l))
}
