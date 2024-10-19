package main

import (
	"log"
	"net"
	"strings"
	"time"
)

func main() {
	// Resolve the broadcast address to send messages.
	broadcastAddr, err := net.ResolveUDPAddr("udp", "255.255.255.255:9933")
	if err != nil {
		log.Fatalf("Failed to resolve broadcast address: %v", err)
	}

	// Listen on a random local UDP port to receive responses.
	localAddr, err := net.ResolveUDPAddr("udp", ":0") // ":0" chooses a random port.
	if err != nil {
		log.Fatalf("Failed to resolve local address: %v", err)
	}

	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		log.Fatalf("Failed to listen on UDP: %v", err)
	}
	defer conn.Close()

	log.Printf("Broadcast sender listening on %s...\n", conn.LocalAddr())

	// Create a buffer to receive responses.
	responseBuf := make([]byte, 256)

	for {
		// Message to broadcast.
		message := []byte("BEGIN tomas")

		// Send the broadcast message.
		_, err := conn.WriteToUDP(message, broadcastAddr)
		if err != nil {
			log.Printf("Error sending broadcast: %v", err)
		} else {
			log.Printf("Broadcast sent: %s", message)
		}

		// Set a read deadline to avoid blocking indefinitely.
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))

		// Listen for a response.
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
				break
			}
		}

		// Wait 2 seconds before sending the next broadcast.
		time.Sleep(2 * time.Second)
	}
}
