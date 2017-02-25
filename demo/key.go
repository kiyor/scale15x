package main

import (
	"encoding/base64"
	"log"
)

const (
	parmaryKeyBase64 = "X/oUj6s/iS7ABlKfpWHgTQ=="
)

func parmaryKey(base64key string) []byte {
	s, err := base64toByte(base64key)
	if err != nil {
		panic(err)
	}
	return s
}

func base64toByte(in string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		log.Println(err.Error())
		return []byte{}, err
	}
	return data, nil
}
