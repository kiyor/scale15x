package main

import (
	"fmt"
)

var (
	version   = "1.0"
	buildtime = ""
)

func Version() string {
	return fmt.Sprint(version + "." + buildtime)
}
