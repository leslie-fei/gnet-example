package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

var lc = net.ListenConfig{
	Control: func(network, address string, c syscall.RawConn) error {
		var opErr error
		if err := c.Control(func(fd uintptr) {
			opErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
		}); err != nil {
			return err
		}
		return opErr
	},
}

func main() {
	server := &http.Server{}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Hello World\n")
	})

	var wg sync.WaitGroup
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		id := i
		go func() {
			defer wg.Done()
			// reuse
			l, err := lc.Listen(context.Background(), "tcp", ":18080")
			if err != nil {
				panic(err)
			}
			fmt.Printf("HTTP Server with ID: %d is running \n", id)
			panic(server.Serve(l))
		}()
	}
	wg.Wait()
}
