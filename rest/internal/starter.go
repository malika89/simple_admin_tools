package internal

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/tal-tech/go-zero/core/proc"
)

func StartHttp(host string, port int, handler http.Handler) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	server := buildHttpServer(addr, handler)
	return start(server, func(srv *http.Server) error {
		return srv.ListenAndServe()
	})
}

func StartHttps(host string, port int, certFile, keyFile string, handler http.Handler) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	server, err := buildHttpsServer(addr, handler, certFile, keyFile)
	if err != nil {
		return err
	}

	return start(server, func(srv *http.Server) error {
		// certFile and keyFile are set in buildHttpsServer
		return srv.ListenAndServeTLS("", "")
	})
}

func buildHttpServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{Addr: addr, Handler: handler}
}

func buildHttpsServer(addr string, handler http.Handler, certFile, keyFile string) (*http.Server, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	config := tls.Config{Certificates: []tls.Certificate{cert}}
	return &http.Server{
		Addr:      addr,
		Handler:   handler,
		TLSConfig: &config,
	}, nil
}

func start(server *http.Server, run func(srv *http.Server) error) error {
	waitForCalled := proc.AddWrapUpListener(func() {
		server.Shutdown(context.Background())
	})
	defer waitForCalled()
	return run(server)
}
