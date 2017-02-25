package main

import (
	"crypto/rand"
	"encoding/base64"
)

func random(i int) string {
	size := i

	rb := make([]byte, size)
	rand.Read(rb)

	rs := base64.StdEncoding.EncodeToString(rb)
	return rs
}
