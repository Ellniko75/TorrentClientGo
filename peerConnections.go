package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

func connectToPeerAndRequestBlockOfFile(ip string, fileIndex int, infoHash []byte, peerID [20]byte, blockOffset int, blockLength int) ([]byte, error) {
	//create the tcp connection
	conn, err := createTcpConnection(ip)
	if err != nil {
		return nil, err
	}

	//send the handshake
	data, err := handleHandshake(infoHash, peerID, &conn)
	if err != nil {
		return nil, createError("connectToPeerAndRequestBlockOfFile()", err.Error())
	}
	printWithColor(Red, fmt.Sprint("el handshake de vuelta respondio"))

	//send the request for the data
	data, err = sendPayload(&conn, fileIndex, blockOffset, blockLength)
	if err != nil {
		return nil, err
	}

	fmt.Print("data Sent")

	return data, nil

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

func handleHandshake(infoHash []byte, peerID [20]byte, connPtr *net.Conn) ([]byte, error) {
	//deserialize the pointer
	conn := *connPtr
	//handle the handshake
	var handshakeMessage bytes.Buffer

	//Write the length (pstrlen)
	if err := binary.Write(&handshakeMessage, binary.BigEndian, byte(19)); err != nil {
		return nil, createError("connectToPeerAndRequestBlockOfFile()", err.Error())
	}
	//Write the protocol (pstr)
	if err := binary.Write(&handshakeMessage, binary.BigEndian, []byte("BitTorrent protocol")); err != nil {
		return nil, createError("connectToPeerAndRequestBlockOfFile()", err.Error())
	}
	//Write the reserved 8 bytes (reserved)
	if err := binary.Write(&handshakeMessage, binary.BigEndian, make([]byte, 8)); err != nil {
		return nil, createError("connectToPeerAndRequestBlockOfFile()", err.Error())
	}
	//Write the hash (info_hash)
	if err := binary.Write(&handshakeMessage, binary.BigEndian, infoHash); err != nil {
		return nil, createError("connectToPeerAndRequestBlockOfFile()", err.Error())
	}
	//Write the peerID (peer_id)
	if err := binary.Write(&handshakeMessage, binary.BigEndian, peerID); err != nil {
		return nil, createError("connectToPeerAndRequestBlockOfFile()", err.Error())
	}

	n, err := conn.Write(handshakeMessage.Bytes())
	if err != nil {
		return nil, createError("connectToPeerAndRequestBlockOfFile()", err.Error())
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

func sendPayload(connPtr *net.Conn, fileIndex int, blockOffset int, blockLength int) ([]byte, error) {
	conn := *connPtr
	//load the payload to send to the peer
	var buff bytes.Buffer

	//Size of the request (Message Length)
	if err := binary.Write(&buff, binary.BigEndian, int32(13)); err != nil {
		return nil, createError("sendPayload() Message Length ", err.Error())
	}
	//indicate that this is a request with the 6 (Message ID)
	if err := binary.Write(&buff, binary.BigEndian, byte(6)); err != nil {
		return nil, createError("sendPayload() Message ID  ", err.Error())
	}
	//The index of the piece being requested. (Piece Index)
	if err := binary.Write(&buff, binary.BigEndian, int32(fileIndex)); err != nil {
		return nil, createError("sendPayload() Piece Index", err.Error())
	}
	//Block Length
	if err := binary.Write(&buff, binary.BigEndian, int32(blockOffset)); err != nil {
		return nil, createError("sendPayload() Block Length", err.Error())
	}
	//Block length
	if err := binary.Write(&buff, binary.BigEndian, int32(blockLength)); err != nil {
		return nil, createError("sendPayload() ", err.Error())
	}

	//send the payload requesting the file
	n, err := conn.Write(buff.Bytes())
	if n == 0 || err != nil {
		return nil, createError("sendPayload() on conn.Write()", err.Error())
	}
	//create the response slice, expecting it to be of length "blockLength" but + 18 because that's the amount of bytes the protocol adds on top
	//of the file
	var response = make([]byte, blockLength+18)
	n, err = conn.Read(response)
	if err != nil {
		return nil, createError("sendPayload() on conn.Write()", err.Error())
	}

	conn.Close()
	//return only the data part of the response
	return response[18:], nil
}
