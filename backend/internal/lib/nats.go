package lib

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

func GetJetStream(natsURL string) (nats.JetStreamContext, *nats.Conn, error) {
	chatMaxAgeStr := Getenv("CHAT_MAX_AGE", "12h")

	chatMaxAge, err := time.ParseDuration(chatMaxAgeStr)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid CHAT_MAX_AGE: %v", err)
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to NATS: %v", err)
	}

	// streams preparation
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("failed to create JetStream: %v", err)
	}

	// add stream for notifications and chat messages, with a retention policy of 24 hours
	streamConf := &nats.StreamConfig{
		Name:     "SESSIONS",
		Subjects: []string{NotiSubjectPrefix + ">", ChatSubjectPrefix + ">"},
		MaxAge:   chatMaxAge,
		Storage:  nats.FileStorage,
	}

	_, err = js.AddStream(streamConf)
	if err != nil {
		if err == nats.ErrStreamNameAlreadyInUse {
			_, err = js.UpdateStream(streamConf)
			if err != nil {
				nc.Close()
				return nil, nil, fmt.Errorf("failed to update sessions stream: %v", err)
			}
		} else {
			nc.Close()
			return nil, nil, fmt.Errorf("failed to create sessions stream: %v", err)
		}
	}

	return js, nc, nil
}
