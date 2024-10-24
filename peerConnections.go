package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

func connectToPeerAndRequestWholePiece(ip string, fileIndex int, infoHash []byte, peerID [20]byte, blockLength int, amountOfBlocks int) ([]byte, error) {
	//create the tcp connection
	conn, err := createTcpConnection(ip)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	//send the handshake
	data, err := handleHandshake(infoHash, peerID, &conn)
	if err != nil {
		return nil, createError("connectToPeerAndRequestBlockOfFile()", err.Error())
	}
	//printWithColor(Red, fmt.Sprint("el handshake de vuelta respondio"))

	wholePiece := []byte{}
	for i := 0; i < amountOfBlocks; i++ {
		blockOffset := blockLength * i

		//send the request for the data
		data, err = requestBlock(&conn, fileIndex, blockOffset, blockLength)
		if err != nil {
			return nil, err
		} else {
			time.Sleep(1 * time.Second)
			wholePiece = append(wholePiece, data...)
		}
	}
	printWithColor(Yellow, fmt.Sprint("Downloaded piece: ", fileIndex))
	//hard to explain, but basically the first requestBlock() petition has a padding of 5 extra bytes that the other consecuently don't have,
	//therefore we just don't send those first 5 bytes as the final complete piece
	return wholePiece[5:], nil
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

func requestBlock(connPtr *net.Conn, fileIndex int, blockOffset int, blockLength int) ([]byte, error) {
	printWithColor(Red, fmt.Sprint("requesting index: ", fileIndex, " block offset: ", blockOffset))
	conn := *connPtr
	//load the payload to send to the peer
	var buff bytes.Buffer

	//Size of the request (Message Length)
	if err := binary.Write(&buff, binary.BigEndian, int32(13)); err != nil {
		return nil, createError("requestBlock() Message Length ", err.Error())
	}
	//indicate that this is a request with the 6 (Message ID)
	if err := binary.Write(&buff, binary.BigEndian, byte(6)); err != nil {
		return nil, createError("requestBlock() Message ID  ", err.Error())
	}
	//The index of the piece being requested. (Piece Index)
	if err := binary.Write(&buff, binary.BigEndian, int32(fileIndex)); err != nil {
		return nil, createError("requestBlock() Piece Index", err.Error())
	}
	//Block Offset
	if err := binary.Write(&buff, binary.BigEndian, int32(blockOffset)); err != nil {
		return nil, createError("requestBlock() Block Length", err.Error())
	}
	//Block length
	if err := binary.Write(&buff, binary.BigEndian, int32(blockLength)); err != nil {
		return nil, createError("requestBlock() ", err.Error())
	}

	//send the payload requesting the file
	n, err := conn.Write(buff.Bytes())
	if n == 0 || err != nil {
		return nil, createError("requestBlock() on conn.Write()", err.Error())
	}
	//response has to be around 20.000 bytes since each block of response is 16kb (16.000 bytes)
	var response = make([]byte, 20000)
	n, err = conn.Read(response)
	if err != nil {
		return nil, createError("requestBlock() on conn.Write()", err.Error())
	}

	fmt.Println("")
	fmt.Println("")
	fmt.Println("")
	fmt.Println("")
	fmt.Println("")
	//return only the data, not the metadata
	return response[13:n], nil
}
