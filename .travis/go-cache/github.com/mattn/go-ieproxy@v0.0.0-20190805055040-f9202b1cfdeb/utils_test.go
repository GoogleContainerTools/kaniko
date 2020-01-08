package ieproxy

import (
	"log"
	"net"
	"net/http"
)

// For testing purposes

func listenAndServeWithClose(addr string, handler http.Handler) (net.Listener, error) {

	var (
		listener net.Listener
		err      error
	)

	srv := &http.Server{Addr: addr, Handler: handler}

	if addr == "" {
		addr = ":http"
	}

	listener, err = net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	go func() {
		err := srv.Serve(listener.(*net.TCPListener))
		if err != nil {
			log.Println("HTTP Server Error - ", err)
		}
	}()

	return listener, nil
}
