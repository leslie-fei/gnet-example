package main

import (
	"bytes"
	"context"
	"errors"
	"runtime"
	"time"

	"github.com/cloudwego/netpoll"
)

func main() {
	listener, err := netpoll.CreateListener("tcp", ":9998")
	if err != nil {
		panic("create netpoll listener failed")
	}

	eventLoop, _ := netpoll.NewEventLoop(
		onReq,
		netpoll.WithReadTimeout(time.Second),
	)
	err = netpoll.SetNumLoops(runtime.NumCPU())
	if err != nil {
		panic(err)
	}

	eventLoop.Serve(listener)
}

var rsp = []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 12\r\n\r\nHello world!")

func onReq(ctx context.Context, connection netpoll.Connection) error {
	r := connection.Reader()
	buf, _ := r.Next(r.Len())
	if !bytes.Contains(buf, []byte("\r\n\r\n")) {
		return errors.New("no delimiter")
	}
	_, _ = connection.Write(rsp)
	return nil
}
