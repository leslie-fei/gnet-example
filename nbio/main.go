package main

import (
	"bytes"
	"fmt"
	"runtime"

	"github.com/lesismal/nbio"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU() * 2)
}

var rsp = []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 12\r\n\r\nHello world!")

func main() {
	engine := nbio.NewEngine(nbio.Config{
		Network:            "tcp",
		Addrs:              []string{":8888"},
		MaxWriteBufferSize: 6 * 1024 * 1024,
		NPoller:            runtime.NumCPU(),
	})

	engine.OnData(func(c *nbio.Conn, data []byte) {
		buf := data
		if !bytes.Contains(buf, []byte("\r\n\r\n")) {
			logging.Errorf("no delimiter")
			return
		}
		_, _ = c.Write(rsp)
	})

	err := engine.Start()
	if err != nil {
		fmt.Printf("nbio.Start failed: %v\n", err)
		return
	}
	defer engine.Stop()

	<-make(chan int)
}
