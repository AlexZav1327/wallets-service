package main

import (
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

func main() {
	port := "8080"

	http.HandleFunc("/time", func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		currentTime := now.Format("02-01-2006 15:04:05")
		fmt.Fprint(w, currentTime)
	})

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
