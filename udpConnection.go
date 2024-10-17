package main

import (
	"fmt"
	"net"
)

func createUdpConnection(url string) (*net.UDPConn, error) {

	// Resolve UDP address
	addr, err := net.ResolveUDPAddr("udp", url)

	if err != nil {
		return nil, err
	}

	// Create UDP connection
	conn, err := net.DialUDP("udp", nil, addr)

	if err != nil {
		fmt.Println("Error creating UDP connection:", err)
		return nil, err
	}
	return conn, nil
}
