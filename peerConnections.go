package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

func connectToPeerAndRequestFile(ip string, fileIndex int, fileHash []byte, peerID [20]byte) error {
	//create the tcp connection
	conn, err := createTcpConnection(ip)
	if err != nil {
		return err
	}

	//send the handshake
	data, err := handleHandshake(fileHash, peerID, &conn)
	if err != nil {
		return createError("connectToPeerAndRequestFile()", err.Error())
	}
	printWithColor(Red, fmt.Sprint("el handshake de vuelta: ", data))

	//send the request for the data
	err = sendPayload(&conn, fileIndex, 0)
	fmt.Print("data Sent")

	return nil

}

// Creates the tcp connection and dials up with the url, for now it's hardcoded to request to the port I know its opened, since I cannot make the port be good
func createTcpConnection(ip string) (net.Conn, error) {
	// Connect to the server
	printWithColor(Yellow, fmt.Sprint(" Attempting to Connect: ", ip))
	conn, err := net.DialTimeout("tcp", ip, 2*time.Second)
	if err != nil {
		return nil, createError("createTcpConnection()", err.Error())
	}
	printWithColor(Yellow, fmt.Sprint(" Connected to: ", ip))
	return conn, nil
}

func handleHandshake(fileHash []byte, peerID [20]byte, connPtr *net.Conn) ([]byte, error) {
	//deserialize the pointer
	conn := *connPtr
	//handle the handshake
	var handshakeMessage bytes.Buffer

	//Write the length
	if err := binary.Write(&handshakeMessage, binary.BigEndian, byte(9)); err != nil {
		return nil, createError("connectToPeerAndRequestFile()", err.Error())
	}
	//Write the protocol
	if err := binary.Write(&handshakeMessage, binary.BigEndian, []byte("BitTorrent")); err != nil {
		return nil, createError("connectToPeerAndRequestFile()", err.Error())
	}
	//Write the reserved 8 bytes
	if err := binary.Write(&handshakeMessage, binary.BigEndian, make([]byte, 8)); err != nil {
		return nil, createError("connectToPeerAndRequestFile()", err.Error())
	}
	//Write the hash
	if err := binary.Write(&handshakeMessage, binary.BigEndian, fileHash); err != nil {
		return nil, createError("connectToPeerAndRequestFile()", err.Error())
	}
	//Write the peerID
	if err := binary.Write(&handshakeMessage, binary.BigEndian, peerID); err != nil {
		return nil, createError("connectToPeerAndRequestFile()", err.Error())
	}

	n, err := conn.Write(handshakeMessage.Bytes())
	if err != nil {
		return nil, createError("connectToPeerAndRequestFile()", err.Error())
	} else {
		printWithColor(Yellow, fmt.Sprint(" Hanshake Bytes sent:", n))
	}

	data := make([]byte, 2048) // Adjust the size as needed
	_, err = conn.Read(data)
	if err != nil {
		return nil, createError("handleHandshake()", err.Error())
	}
	return data, nil
}

func sendPayload(connPtr *net.Conn, fileIndex int, beginIndex int) error {
	conn := *connPtr
	//load the payload to send to the peer
	var buff bytes.Buffer
	//Size of the request
	if err := binary.Write(&buff, binary.BigEndian, byte(9)); err != nil {
		return createError("sendPayload()", err.Error())

	}
	//indicate that this is a request with the 6
	if err := binary.Write(&buff, binary.BigEndian, byte(6)); err != nil {
		return createError("sendPayload()", err.Error())
	}
	//file index
	if err := binary.Write(&buff, binary.BigEndian, int32(fileIndex)); err != nil {
		return createError("sendPayload()", err.Error())
	}
	//begin index
	if err := binary.Write(&buff, binary.BigEndian, int32(beginIndex)); err != nil {
		return createError("sendPayload()", err.Error())
	}

	//send the payload requesting the file
	n, err := conn.Write(buff.Bytes())
	if n == 0 || err != nil {
		return createError("sendPayload() on conn.Write()", err.Error())
	}

	var response = make([]byte, 10000)
	n, err = conn.Read(response)
	if n == 0 {
		return createError("sendPayload() on conn.Write()", "Response is 0 bytes")
	}
	if err != nil {
		return createError("sendPayload() on conn.Write()", err.Error())
	}

	conn.Close()

	return nil
}

func listenConnections() ([]byte, error) {
	listener, err := net.Listen("tcp", ":6881")
	if err != nil {
		return nil, createError("connectToPeerAndRequestFile()", err.Error())
	}
	fmt.Println("listening...")
	for {
		conn, err := listener.Accept()
		printWithColor(Red, "CONNECTIONS ARRIVED LMAOOOOOOOOOOO")
		if err != nil {
			return nil, createError("listener.Accept() on listenConnections()", err.Error())
		}
		response, err := readIncomingMessages(conn)
		if err != nil {
			printWithColor(Red, fmt.Sprint("la cagamo", err.Error()))
			return nil, err
		} else {
			printWithColor(Yellow, fmt.Sprint("WE DID IT, THE MESSAGE IS HERE: ", response))
			return response, nil
		}
	}
}

func readIncomingMessages(conn net.Conn) ([]byte, error) {
	buffer := make([]byte, 500000)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Println("Error reading from connection:", err)
			return nil, createError("handleConnections()", err.Error())
		}
		if n == 0 {
			return buffer, nil
		}
		fmt.Println("Received:", string(buffer[:n]))
	}
}
