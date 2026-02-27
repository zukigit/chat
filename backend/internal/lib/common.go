package lib

import (
	"log"
	"os"
)

func Getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

var (
	InfoLog  = log.New(os.Stdout, "[INFO] ", log.LstdFlags)
	ErrorLog = log.New(os.Stderr, "[ERROR] ", log.LstdFlags)
	WarnLog  = log.New(os.Stdout, "[WARN] ", log.LstdFlags)
)
