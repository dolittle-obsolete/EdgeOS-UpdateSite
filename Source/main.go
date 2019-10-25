package main

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	fileSystem := http.Dir("/www")
	server := newCompletionFileServer(fileSystem)

	instrumented := instrumentSwupdServer(server)

	go func() {
		log.Println("Serving metrics on *:9700/metrics")
		err := http.ListenAndServe(":9700", promhttp.Handler())
		log.Println("Metrics server failed with: ", err)
	}()

	log.Println("Serving updates on *:80/ from", fileSystem)
	err := http.ListenAndServe(":80", instrumented)
	log.Fatalln("Updates server failed with:", err)
}
