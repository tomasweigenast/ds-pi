package shared

import (
	"crypto/rand"
	"fmt"
	"net"
)

const MASTER_PORT = 9999

func GetIPv4() (net.IP, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range interfaces {
		// Ignore down or loopback interfaces
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ip := ipNet.IP
			// Check if it's an IPv4 address and not a loopback
			if ip.To4() != nil && !ip.IsLoopback() {
				return ip, nil
			}
		}
	}

	return nil, fmt.Errorf("no non-loopback IPv4 address found")
}

var chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

// RandomString generates a random string
func RandomString() string {
	ll := len(chars)
	b := make([]byte, 8)
	rand.Read(b)
	for i := 0; i < 8; i++ {
		b[i] = chars[int(b[i])%ll]
	}
	return string(b)
}
