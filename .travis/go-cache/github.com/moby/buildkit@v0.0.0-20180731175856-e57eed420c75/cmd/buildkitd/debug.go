package main

import (
	"expvar"
	"net"
	"net/http"
	"net/http/pprof"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/trace"
)

func setupDebugHandlers(addr string) error {
	m := http.NewServeMux()
	m.Handle("/debug/vars", expvar.Handler())
	m.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	m.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	m.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	m.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	m.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	// TODO: reenable after golang.org/x/net update
	// m.Handle("/debug/requests", http.HandlerFunc(trace.Traces))
	// m.Handle("/debug/events", http.HandlerFunc(trace.Events))

	// setting debugaddr is opt-in. permission is defined by listener address
	trace.AuthRequest = func(_ *http.Request) (bool, bool) {
		return true, true
	}

	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	logrus.Debugf("debug handlers listening at %s", addr)
	go http.Serve(l, m)
	return nil
}
