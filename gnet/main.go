package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"sync"

	"github.com/panjf2000/gnet/v2"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

type httpServer struct {
	gnet.BuiltinEventEngine

	addr      string
	multicore bool
	eng       gnet.Engine
}

func (hs *httpServer) OnBoot(eng gnet.Engine) gnet.Action {
	hs.eng = eng
	log.Printf("echo server with multi-core=%t is listening on %s\n", hs.multicore, hs.addr)
	return gnet.None
}

func (hs *httpServer) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	return nil, gnet.None
}

var rsp = []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 12\r\n\r\nHello world!")

var bufferPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

func (hs *httpServer) OnTraffic(c gnet.Conn) gnet.Action {
	buffer := bufferPool.Get().(*bytes.Buffer)
	defer func() {
		buffer.Reset()
		bufferPool.Put(buffer)
	}()
	_, _ = c.WriteTo(buffer)
	if !bytes.Contains(buffer.Bytes(), []byte("\r\n\r\n")) {
		logging.Errorf("no delimiter")
		return gnet.Close
	}
	_, _ = c.Write(rsp)
	return gnet.None
}

func main() {
	var port int
	var multicore bool

	// Example command: go run main.go --port 8080 --multicore=true
	flag.IntVar(&port, "port", 8081, "server port")
	flag.BoolVar(&multicore, "multicore", true, "multicore")
	flag.Parse()

	hs := &httpServer{addr: fmt.Sprintf("tcp://:%d", port), multicore: multicore}

	// Start serving!
	log.Println("server exits:", gnet.Run(hs, hs.addr, gnet.WithMulticore(multicore)))
}
