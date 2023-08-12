package service

import "fmt"

type AccessData struct{}

func (*AccessData) ShowCurrentAccessData(ip string, time string) string {
	return fmt.Sprintf("Your IP address is: %s\tCurrent date and time is: %s", ip, time)
}

func (*AccessData) ShowPreviousAccessData(data map[string][]string) string {
	var convertedData string

	for ip, time := range data {
		var convertedTime string
		for _, v := range time {
			convertedTime += fmt.Sprintf("%s; ", v)
		}

		convertedData += fmt.Sprintf("IP address: %s\tDate and time: %s\n", ip, convertedTime[:len(convertedTime)-2])
	}

	return convertedData
}
