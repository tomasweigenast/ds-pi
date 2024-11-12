package discover

import (
	"fmt"
	"log"
	"net"
	"strings"
)

type Discover struct {
	addr net.UDPAddr
	conn *net.UDPConn

	onDiscover func(*net.UDPAddr, string) bool
}

func NewDiscover(port int) Discover {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(fmt.Errorf("unable to resolve udp address from :%d. %s", port, err))
	}
	return Discover{
		addr: *addr,
	}
}

// OnDiscover is called when a new worker is registered
func (d *Discover) OnDiscover(f func(remoteAddr *net.UDPAddr, name string) bool) {
	d.onDiscover = f
}

// Begin start the discover service searching for workers who want to join
// the calculation. It starts a simple UDP server at broadcast IP.
// This method spawns a new goroutine so it does not block.
func (d *Discover) Begin() {
	if d.onDiscover == nil {
		panic("OnDiscover not set")
	}

	go d.begin()
}

// Stop stops the service
func (d *Discover) Stop() {
	if d.conn != nil {
		d.conn.Close()
		d.conn = nil
	}
}

func (d *Discover) begin() {
	conn, err := net.ListenUDP("udp", &d.addr)
	if err != nil {
		panic(fmt.Errorf("unable to start UDP discovering service: %s", err))
	}

	d.conn = conn
	log.Printf("Discover service started on broadcast at %s", d.addr.String())
	d.handleConnection()
}

func (d *Discover) handleConnection() {
	buffer := make([]byte, 24)
	response := make([]byte, 0, 24)
	for {
		log.Println("Checking for new connections...")
		n, remoteAddr, err := d.conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Error reading from UDP: %s", err)
			continue
		}

		messageParts := strings.Split(strings.TrimSpace(string(buffer[:n])), " ")
		if len(messageParts) != 2 {
			continue
		}

		command := messageParts[0]

		if command == keyword_BEGIN {
			name := messageParts[1]
			log.Printf("Received BEGIN from %s. Name: %s", remoteAddr, name)

			// clear response buffer before set new values
			buffer = buffer[:0]

			// check onDiscover
			if d.onDiscover(remoteAddr, name) {
				response = append(response, keyword_OK...)
				response = append(response, ' ')
				response = append(response, []byte(name)...)
				log.Printf("Accepted %s (%s).", remoteAddr, name)
			} else {
				log.Printf("Rejected %s (%s).", remoteAddr, name)
				response = append(response, keyword_REJECT...)
			}

			_, err := d.conn.WriteTo(response, remoteAddr)
			if err != nil {
				log.Printf("Error sending response: %s", err)
			} else {
				log.Printf("Response sent %s", remoteAddr)
			}
		}
	}
}
