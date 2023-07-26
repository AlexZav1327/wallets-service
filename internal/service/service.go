package service

import "fmt"

type AccessData struct {
	data map[string][]string
}

func (a *AccessData) SaveAccessData(ip string, time string) {
	if a.data == nil {
		a.data = make(map[string][]string)
	}

	_, ok := a.data[ip]

	if ok {
		a.data[ip] = append(a.data[ip], time)
	} else {
		a.data[ip] = []string{time}
	}
}

func (a *AccessData) ShowCurrentAccessData(ip string, time string) string {
	return fmt.Sprintf("Your IP address is: %s\tCurrent date and time is: %s", ip, time)
}

func (a *AccessData) ShowPreviousAccessData() string {
	var convertedData string

	for ip, time := range a.data {
		var convertedTime string
		for _, v := range time {
			convertedTime += fmt.Sprintf("%s; ", v)
		}

		convertedData += fmt.Sprintf("IP address: %s\tDate and time: %s", ip, convertedTime[:len(convertedTime)-2])
	}

	return convertedData
}
