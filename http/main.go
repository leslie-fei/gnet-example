package main

import (
	"bytes"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	cxstrconv "github.com/cloudxaas/gostrconv"
	"github.com/leslie-fei/gnettls"

	//    cxsysinfomem "github.com/cloudxaas/gosysinfo/mem"
	"github.com/evanphx/wildcat"
	"github.com/leslie-fei/gnettls/tls"
	"github.com/panjf2000/gnet/v2"
	"github.com/valyala/bytebufferpool"
)

var (
	errMsg         = "Internal Server Error"
	errMsgBytes    = []byte(errMsg)
	now            atomic.Value
	bufferPool     bytebufferpool.Pool
	responseHeader = []byte("HTTP/1.1 200 OK\r\nServer: gnet\r\nContent-Type: text/plain\r\nDate: ")
)

type httpServer struct {
	gnet.BuiltinEventEngine
	addr      string
	multicore bool
	eng       gnet.Engine
}

type httpCodec struct {
	parser *wildcat.HTTPParser
	buf    *bytebufferpool.ByteBuffer // Main buffer reused for all I/O operations
}

type combinedContext struct {
	httpCodec *httpCodec
}

func updateCurrentTime() {
	now.Store(time.Now().Format(time.RFC1123))
}

func (hc *httpCodec) appendResponse(body []byte) {
	updateCurrentTime() // Update time only when responding
	hc.buf.Write(responseHeader)
	hc.buf.WriteString(now.Load().(string))
	hc.buf.WriteString("\r\nContent-Length: ")
	hc.buf.WriteString(cxstrconv.Inttoa(len(body)))
	hc.buf.WriteString("\r\n\r\n")
	hc.buf.Write(body)
}

func (hs *httpServer) OnBoot(eng gnet.Engine) gnet.Action {
	hs.eng = eng
	log.Printf("HTTP server with multi-core=%t is listening on %s\n", hs.multicore, hs.addr)
	return gnet.None
}

func (hs *httpServer) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	hc := &httpCodec{
		parser: wildcat.NewHTTPParser(),
		buf:    bytebufferpool.Get(),
	}
	c.SetContext(&combinedContext{
		httpCodec: hc,
	})
	return nil, gnet.None
}

func (hs *httpServer) OnClose(c gnet.Conn, err error) gnet.Action {
	ctx, ok := c.Context().(*combinedContext)
	if ok && ctx.httpCodec != nil {
		bufferPool.Put(ctx.httpCodec.buf)
	}
	return gnet.None
}

func route(hc *httpCodec) {
	var response []byte
	if bytes.Equal(hc.parser.Method, []byte("GET")) {
		switch {
		case bytes.Equal(hc.parser.Path, []byte("/hello")):
			response = []byte("Hello, World!")
		case bytes.Equal(hc.parser.Path, []byte("/time")):
			response = []byte("Current Time: " + now.Load().(string))
		default:
			response = []byte("404 Not Found")
		}
	} else {
		response = []byte("405 Method Not Allowed")
	}
	hc.appendResponse(response)
}

func (hs *httpServer) OnTraffic(c gnet.Conn) gnet.Action {
	ctx, ok := c.Context().(*combinedContext)
	if !ok || ctx.httpCodec == nil {
		return gnet.Close
	}
	hc := ctx.httpCodec

	// check http packet is completed
	peeked, _ := c.Peek(c.InboundBuffered())
	if !bytes.Contains(peeked, []byte("\r\n\r\n")) {
		// data not enough do it next round
		return gnet.None
	}

	hc.buf.Reset()
	data, _ := c.Next(-1)
	for {
		headerOffset, err := hc.parser.Parse(data)
		if err != nil {
			c.Write(errMsgBytes)
			return gnet.Close
		}
		route(hc)
		bodyLen := int(hc.parser.ContentLength())
		if bodyLen == -1 {
			bodyLen = 0
		}
		data = data[headerOffset+bodyLen:]
		if len(data) == 0 {
			break //this is pipelined requests, do not use this feature
		}
	}
	c.Write(hc.buf.B)
	return gnet.None
}

func mustLoadCertificate() tls.Certificate {
	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		log.Fatalf("Failed to load server certificate: %v", err)
	}
	return cert
}

func main() {
	go func() {
		hs := &httpServer{
			addr:      fmt.Sprintf("tcp://:%d", 8080),
			multicore: true,
		}

		options := []gnet.Option{
			gnet.WithMulticore(true),
			gnet.WithTCPKeepAlive(time.Minute * 5),
			gnet.WithReusePort(true),
		}

		log.Fatal(gnet.Run(hs, hs.addr, options...))
	}()

	go func() {
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{mustLoadCertificate()},
		}

		hs := &httpServer{
			addr:      fmt.Sprintf("tcp://:%d", 8443),
			multicore: true,
		}

		options := []gnet.Option{
			gnet.WithMulticore(true),
			gnet.WithTCPKeepAlive(time.Minute * 5),
			gnet.WithReusePort(true),
		}

		log.Fatal(gnettls.Run(hs, hs.addr, tlsConfig, options...))
	}()

	select {}
}
