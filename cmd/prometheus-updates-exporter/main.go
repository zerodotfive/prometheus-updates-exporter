package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/zerodotfive/prometheus-updates-exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/pflag"
)

func RootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "ok")
}

func main() {
	var listen string
	listen_def := "0.0.0.0:9028"
	pflag.StringVar(
		&listen,
		"listen",
		listen_def,
		"Listen address. Env LISTEN also can be used.",
	)

	if listen == listen_def && len(os.Getenv("LISTEN")) > 0 {
		listen = os.Getenv("LISTEN")
	}

	log.Printf("Statring updates exporter.")

	updatesExporter, err := exporter.CreateUpdatesExporter()
	if err != nil {
		log.Fatal(err)
	}

	prometheus.MustRegister(updatesExporter)

	go updatesExporter.Run(context.Background())

	http.HandleFunc("/", RootHandler)
	http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(listen, nil); err != nil {
		log.Fatalf("Can't bind on %v", listen)
	}
}
