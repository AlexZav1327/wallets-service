package service

import (
	"fmt"
	"net/http"
	"time"
)

type VisitInfo struct {
	timeVisit string
	ipAddress string
}

func (v *VisitInfo) ShowVisitInfo(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	currentTime := now.Format("02-01-2006 15:04:05")

	fmt.Fprintf(w, "Current date and time is: %s\n\n", currentTime)
	fmt.Fprintf(w, "Your IP address is: %s", r.RemoteAddr)

	v.timeVisit += fmt.Sprintf("%s\n", currentTime)
	v.ipAddress += fmt.Sprintf("%s\n", r.RemoteAddr)
}

func (v *VisitInfo) ShowPrevVisitInfo(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintf(w, "Previous visits date and time were:\n%s\n\n", v.timeVisit)
	fmt.Fprintf(w, "Previous user's IP addresses were:\n%s", v.ipAddress)
}
