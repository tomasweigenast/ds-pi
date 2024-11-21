package shared

import (
	"crypto/rand"
	"fmt"
	"net"
	"runtime"
	"strings"
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

// PrintMemUsage outputs the current, total and OS memory being used. As well as the number
// of garage collection cycles completed.
func PrintMemUsage(variables map[string]any) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	buf := strings.Builder{}
	buf.WriteString("---------------------------------- Memory Stats ------------------------------------\n")
	buf.WriteString(fmt.Sprintf("%-30s %-30s %-30s\n",
		fmt.Sprintf("Alloc = %s", formatBytes(m.Alloc)), fmt.Sprintf("TotalAlloc = %s", formatBytes(m.TotalAlloc)), fmt.Sprintf("Freed = %s", formatBytes(m.TotalAlloc-m.Alloc))))

	buf.WriteString(fmt.Sprintf("%-30s %-30s %-30s\n",
		fmt.Sprintf("Sys = %s", formatBytes(m.Sys)), fmt.Sprintf("NumGC = %s", formatBytes(uint64(m.NumGC))), fmt.Sprintf("Mallocs = %s", formatBytes(m.Mallocs))))

	if len(variables) > 0 {
		buf.WriteString("\nVariables:\n")
		totalSize := uint64(0)
		for name, variable := range variables {
			size := SizeOf(variable)
			if size > 0 {
				totalSize += uint64(size)
				buf.WriteString(fmt.Sprintf("\t%s = %s\n", name, formatBytes(uint64(size))))
			}
		}
		buf.WriteString(fmt.Sprintf("\tTotal = %s\n", formatBytes(uint64(totalSize))))
	}
	buf.WriteString("------------------------------------------------------------------------------------\n")

	fmt.Print(buf.String())
}

func formatBytes(bytes uint64) string {
	const (
		KB = 1 << 10 // 1024
		MB = 1 << 20 // 1024 * 1024
		GB = 1 << 30 // 1024 * 1024 * 1024
		TB = 1 << 40 // 1024 * 1024 * 1024 * 1024
		PB = 1 << 50 // 1024 * 1024 * 1024 * 1024 * 1024
	)

	switch {
	case bytes >= PB:
		return fmt.Sprintf("%.2f PB", float64(bytes)/float64(PB))
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func MapArray[T, U any](ts []T, f func(T) U) []U {
	us := make([]U, len(ts))
	for i := range ts {
		us[i] = f(ts[i])
	}
	return us
}
