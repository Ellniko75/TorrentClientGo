package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"time"
)

func createTcpConnection(url string) (net.Conn, error) {
	// Connect to the server
	printWithColor(Yellow, fmt.Sprint(" Connecting to: ", url))
	conn, err := net.DialTimeout("tcp", url, 10*time.Second)
	if err != nil {
		return nil, createError("createTcpConnection()", err.Error())
	}

	return conn, nil
}

func connectToPeerAndRequestFile(ip string, fileIndex int, fileHash [20]byte, peerID [20]byte) ([]byte, error) {
	//create the tcp connection
	conn, err := createTcpConnection(ip)
	if err != nil {
		return nil, err
	}

	//handle the handshake
	var handshakeMessage bytes.Buffer
	//Write the protocol
	if err = binary.Write(&handshakeMessage, binary.BigEndian, []byte("BitTorrent protocol")); err != nil {
		return nil, createError("connectToPeerAndRequestFile()", err.Error())
	}
	//Write the reserved 8 bytes
	if err = binary.Write(&handshakeMessage, binary.BigEndian, make([]byte, 8)); err != nil {
		return nil, createError("connectToPeerAndRequestFile()", err.Error())
	}
	//Write the hash
	if err = binary.Write(&handshakeMessage, binary.BigEndian, fileHash); err != nil {
		return nil, createError("connectToPeerAndRequestFile()", err.Error())
	}
	//Write the peerID
	if err = binary.Write(&handshakeMessage, binary.BigEndian, peerID); err != nil {
		return nil, createError("connectToPeerAndRequestFile()", err.Error())
	}
	//send the handshake
	_, err = conn.Write(handshakeMessage.Bytes())
	if err != nil {
		return nil, createError("connectToPeerAndRequestFile()", err.Error())
	}
	// Receive handshake response
	response := make([]byte, 68)
	_, err = conn.Read(response)
	if err != nil {
		return nil, createError("connectToPeerAndRequestFile() EL PUTO NO QUIERE HANDSHAKEAR", err.Error())
	}

	//load the payload to send to the peer
	var buff bytes.Buffer
	//Size of the request
	if err = binary.Write(&buff, binary.BigEndian, byte(9)); err != nil {
		log.Println(err)
	}
	//indicate that this is a request with the 6
	if err = binary.Write(&buff, binary.BigEndian, byte(6)); err != nil {
		log.Println(err)
	}
	//file index
	if err = binary.Write(&buff, binary.BigEndian, int32(fileIndex)); err != nil {
		log.Println(err)
	}
	//begin index
	if err = binary.Write(&buff, binary.BigEndian, int32(0)); err != nil {
		log.Println(err)
	}

	//send the payload requesting the file
	n, err := conn.Write(buff.Bytes())
	if n == 0 || err != nil {
		return nil, createError("connectToPeerAndRequestFile() on conn.Write()", err.Error())
	}

	//read the data
	data := []byte{}
	n, err = conn.Read(data)
	if n == 0 || err != nil {
		return nil, createError("connectToPeerAndRequestFile() on conn.Read()", err.Error())
	}

	return data, nil
}
