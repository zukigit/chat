package lib

import (
	"io"
	"log"
	"net/http"
	"strings"
)

func GetPublicIP() string {
	resp, err := http.Get(Getenv("CHAT_GATEWAY_PUBLIC_IP_URL", "https://api.ipify.org"))
	if err != nil {
		log.Printf("could not get public IP: %v", err)
		return "unknown"
	}
	defer resp.Body.Close()

	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("could not read public IP response: %v", err)
		return "unknown"
	}

	return strings.TrimSpace(string(ip))
}
