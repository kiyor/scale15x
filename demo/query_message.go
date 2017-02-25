package main

import (
	"encoding/json"
	"time"
)

type Message struct {
	Body     []byte
	SendTime time.Time
	AckTime  time.Time
	From     string
	FromSrc  string
	Env      *Env
}

func (agent *Agent) NewMessage() Message {
	return Message{
		From:    agent.serf.LocalMember().Name,
		FromSrc: agent.serf.LocalMember().Addr.String(),
		Env:     newEnv(),
	}
}

func (m Message) String() string {
	return string(m.Body)
}

func DecodeMessage(b []byte) (m Message) {
	ErrLog(json.Unmarshal(b, &m))
	return
}

func EncodeMessage(m Message) (b []byte) {
	return Must(json.Marshal(m)).([]byte)
}
