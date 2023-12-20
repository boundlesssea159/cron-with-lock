package cron_with_lock

import (
	"net"
	"runtime"
	"strconv"
	"strings"
)

func GetGoroutineId() (int64, error) {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	stk := strings.TrimPrefix(string(buf[:n]), "goroutine")
	idField := strings.Fields(stk)[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		return -1, err
	}
	return int64(id), nil
}

func IpV4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "0.0.0.0"
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "0.0.0.0"
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					return ipnet.IP.String()
				}
			}
		}
	}
	return "0.0.0.0"
}
