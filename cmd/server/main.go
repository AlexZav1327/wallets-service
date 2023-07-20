package main

import (
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

func showInfo(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	currentTime := now.Format("02-01-2006 15:04:05")
	fmt.Fprintf(w, "Current time: %s\n", currentTime)

	ipAddress := r.RemoteAddr
	fmt.Fprintf(w, "Your IP address is: %s", ipAddress)
}

func main() {
	port := "8080"

	http.HandleFunc("/info", showInfo)

	fmt.Printf("Server is running on http://localhost:%s\n", port)

	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.WithFields(log.Fields{
			"package":  "main",
			"function": "http.ListenAndServe",
			"error":    err,
		}).Panic("Error starting the server")

		return
	}
}
