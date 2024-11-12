package connect

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"ds-pi.com/master/shared"
)

func Connect(workerName string) net.IP {
	broadcastAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("255.255.255.255:%d", shared.DISCOVER_PORT))
	if err != nil {
		log.Fatalf("Failed to resolve broadcast address: %v", err)
	}

	localAddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		log.Fatalf("Failed to resolve local address: %v", err)
	}

	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		log.Fatalf("Failed to listen on UDP: %v", err)
	}
	defer conn.Close()

	log.Printf("Broadcast sender listening on %s...\n", conn.LocalAddr())

	responseBuf := make([]byte, 256)

	tryCount := 0
	for {
		message := []byte(fmt.Sprintf("BEGIN %s", workerName))

		_, err := conn.WriteToUDP(message, broadcastAddr)
		if err != nil {
			log.Printf("Error sending broadcast: %v", err)
		} else {
			log.Printf("Broadcast sent: %s", message)
		}

		conn.SetReadDeadline(time.Now().Add(1 * time.Minute))

		n, addr, err := conn.ReadFromUDP(responseBuf)
		if err != nil {
			log.Println("No response received:", err)
		} else {
			response := string(responseBuf[:n])
			log.Printf("Received response: '%s' from %s\n", response, addr)

			cmd := strings.Split(response, " ")
			if len(cmd) != 2 {
				continue
			}

			if cmd[0] == "OK" {
				log.Printf("Server is at %s", addr)
				return addr.IP
			}
		}

		time.Sleep(2 * time.Second)
		tryCount++

		if tryCount > 10 {
			log.Fatalf("Unable to connect to master after 10 tries.")
		}
	}
}
