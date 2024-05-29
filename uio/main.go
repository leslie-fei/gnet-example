package main

import (
	"bytes"
	"flag"
	"fmt"
	"sync"

	"github.com/panjf2000/gnet/v2/pkg/logging"
	"github.com/urpc/uio"
)

var rsp = []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 12\r\n\r\nHello world!")

func main() {

	var port int
	var loops int
	flag.IntVar(&port, "port", 9527, "server port")
	flag.IntVar(&loops, "loops", 0, "server loops")
	flag.Parse()

	var events uio.Events
	events.Pollers = loops
	events.Addrs = []string{fmt.Sprintf("tcp://:%d", port)}
	var bufferPool = sync.Pool{
		New: func() interface{} {
			return &bytes.Buffer{}
		},
	}
	events.OnData = func(c uio.Conn) error {
		buffer := bufferPool.Get().(*bytes.Buffer)
		defer func() {
			buffer.Reset()
			bufferPool.Put(buffer)
		}()
		_, _ = c.WriteTo(buffer)
		if !bytes.Contains(buffer.Bytes(), []byte("\r\n\r\n")) {
			logging.Errorf("no delimiter")
			return nil
		}
		_, _ = c.Write(rsp)
		return nil
	}

	fmt.Printf("uio echo server with loop=%d is listening on %s\n", events.Pollers, events.Addrs[0])

	if err := events.Serve(); nil != err {
		panic(fmt.Errorf("uio server exit, error: %v", err))
	}
}
