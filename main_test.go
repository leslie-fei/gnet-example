package main

import (
	tls2 "crypto/tls"
	"io"
	"log"
	"net/http"
	"testing"
	"time"
)

func TestGOTLS(t *testing.T) {
	go clientToCall("https://127.0.0.1:444")
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("Hello world!"))
	})

	certFile := "server.crt" // 证书文件路径
	keyFile := "server.key"  // 私钥文件路径

	err := http.ListenAndServeTLS(":444", certFile, keyFile, nil)
	if err != nil {
		log.Fatal("ListenAndServeTLS: ", err)
	}
}

func TestGnetTLS(t *testing.T) {
	go clientToCall("https://127.0.0.1:443")
	runHTTPServer()
}

func clientToCall(url string) {
	time.Sleep(time.Second)
	// new a http client to call
	var httpClient = &http.Client{
		Timeout: 5 * time.Minute,
		Transport: &http.Transport{
			TLSClientConfig: &tls2.Config{
				InsecureSkipVerify: true,
				MaxVersion:         tls2.VersionTLS12,
			},
		},
	}
	//httpClient := http.DefaultClient

	call := func() {
		rsp, err := httpClient.Get(url)
		if err != nil {
			log.Printf("client call error: %v\n", err)
			return
		}
		defer rsp.Body.Close()
		data, err := io.ReadAll(rsp.Body)
		if err != nil {
			log.Fatalf("read data error: %v\n", err)
		}
		if len(data) != 12 {
			log.Fatalf("invalid data length: %d\n", len(data))
		}
		log.Printf("http client call success, code: %d, data: %s\n", rsp.StatusCode, data)
	}

	for i := 0; i < 1; i++ {
		call()
	}
}
