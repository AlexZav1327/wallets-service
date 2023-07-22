package service

import (
	"fmt"
	"net/http"
	"time"
)

type VisitInfo struct {
	timeVisit   string
	ipAddress   string
	summaryInfo string
}

func (v *VisitInfo) ShowVisitInfo(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	currentTime := now.Format("02-01-2006 15:04:05")

	fmt.Fprintf(w, "Current date and time is: %s\n\n", currentTime)
	fmt.Fprintf(w, "Your IP address is: %s", r.RemoteAddr)

	v.timeVisit = fmt.Sprintf("Date and time: %s", currentTime)
	v.ipAddress = fmt.Sprintf("IP address: %s", r.RemoteAddr)
	v.summaryInfo += fmt.Sprintf("%s\t%s\n", v.timeVisit, v.ipAddress)
}

func (v *VisitInfo) ShowPrevVisitInfo(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintf(w, "Previous visits information:\n\n%s\n", v.summaryInfo)
}
