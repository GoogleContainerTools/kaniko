package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/google/go-containerregistry/pkg/registry"
)

var port = flag.Int("port", 1338, "port to run registry on")

func main() {
	flag.Parse()
	s := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: registry.New(),
	}
	log.Fatal(s.ListenAndServe())
}
