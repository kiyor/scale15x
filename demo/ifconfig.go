package main

import (
	"github.com/hashicorp/memberlist"
	"net"
)

func GetPublicIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback then return
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil && !memberlist.IsPrivateIP(ipnet.IP.String()) {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
