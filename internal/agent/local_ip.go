package agent

import (
	"fmt"
	"net"
)

func localIPFor(serverHost string) (net.IP, error) {
	// The port number here is arbitrary; UDP dial doesn't actually send
	// packets or connect.
	conn, err := net.Dial("udp4", net.JoinHostPort(serverHost, "1"))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	udpAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return nil, fmt.Errorf("unexpected local addr type %T", conn.LocalAddr())
	}

	return udpAddr.IP, nil
}
