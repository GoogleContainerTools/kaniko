// Copyright 2018, OpenCensus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ochttp_test

import (
	"log"
	"net/http"

	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/plugin/ochttp/propagation/b3"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

func ExampleTransport() {
	if err := view.Register(
		// Register to a few default views.
		ochttp.ClientRequestCountByMethod,
		ochttp.ClientResponseCountByStatusCode,
		ochttp.ClientLatencyView,

		// Register to a custom view.
		&view.View{
			Name:        "httpclient_latency_by_hostpath",
			TagKeys:     []tag.Key{ochttp.Host, ochttp.Path},
			Measure:     ochttp.ClientLatency,
			Aggregation: ochttp.DefaultLatencyDistribution,
		},
	); err != nil {
		log.Fatal(err)
	}

	client := &http.Client{
		Transport: &ochttp.Transport{},
	}
	_ = client // use client to perform requests
}

var usersHandler http.Handler

func ExampleHandler() {
	// Enables OpenCensus for the default serve mux.
	// By default, B3 propagation is used.
	http.Handle("/users", usersHandler)
	log.Fatal(http.ListenAndServe("localhost:8080", &ochttp.Handler{}))
}

func ExampleHandler_mux() {
	mux := http.NewServeMux()
	mux.Handle("/users", usersHandler)

	log.Fatal(http.ListenAndServe("localhost:8080", &ochttp.Handler{
		Handler:     mux,
		Propagation: &b3.HTTPFormat{},
	}))
}
