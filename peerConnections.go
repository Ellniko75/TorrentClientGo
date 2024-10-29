package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"time"
)

// Connects to the peer anr requests the whole file, block by block
func connectToPeerAndRequestWholePiece(conn *Connection, fileIndex int, blockLength int, amountOfBlocks int) ([]byte, error) {

	wholePiece := []byte{}
	for i := 0; i < amountOfBlocks; i++ {
		blockOffset := blockLength * i

		//send the request for the data
		data, err := requestBlock(conn.Conn, fileIndex, blockOffset, blockLength)
		if err != nil {
			return nil, err
		} else {
			//time.Sleep(1 * time.Second)
			wholePiece = append(wholePiece, data...)
		}
	}

	//for all the other pieces the peers do not send that 5 bytes, so que return the whole piece
	return wholePiece, nil
}
func initiatePeerConnection(ip string, infoHash []byte, peerId [20]byte) (net.Conn, error) {
	//create the connection based on the IP
	connection, err := createTcpConnection(ip)
	if err != nil {
		return nil, err
	}
	_, err = handleHandshake(infoHash, peerId, connection)
	if err != nil {
		return nil, err
	}

	return connection, nil
}

// Creates the tcp connection and dials up with the url, for now it's hardcoded to request to the port I know its opened, since I cannot make the port be good
func createTcpConnection(ip string) (net.Conn, error) {
	// Connect to the server
	//printWithColor(Yellow, fmt.Sprint(" Attempting to Connect: ", ip))
	conn, err := net.DialTimeout("tcp", ip, 2*time.Second)
	if err != nil {
		return nil, createError("createTcpConnection()", err.Error())
	}
	//printWithColor(Yellow, fmt.Sprint(" Connected to: ", ip))
	return conn, nil
}

func handleHandshake(infoHash []byte, peerID [20]byte, conn net.Conn) ([]byte, error) {
	//deserialize the pointer

	//handle the handshake
	var handshakeMessage bytes.Buffer

	//Write the length (pstrlen)
	if err := binary.Write(&handshakeMessage, binary.BigEndian, byte(19)); err != nil {
		return nil, createError("handleHandshake()", err.Error())
	}
	//Write the protocol (pstr)
	if err := binary.Write(&handshakeMessage, binary.BigEndian, []byte("BitTorrent protocol")); err != nil {
		return nil, createError("handleHandshake()", err.Error())
	}
	//Write the reserved 8 bytes (reserved)
	if err := binary.Write(&handshakeMessage, binary.BigEndian, make([]byte, 8)); err != nil {
		return nil, createError("handleHandshake()", err.Error())
	}
	//Write the hash (info_hash)
	if err := binary.Write(&handshakeMessage, binary.BigEndian, infoHash); err != nil {
		return nil, createError("handleHandshake()", err.Error())
	}
	//Write the peerID (peer_id)
	if err := binary.Write(&handshakeMessage, binary.BigEndian, peerID); err != nil {
		return nil, createError("handleHandshake()", err.Error())
	}

	n, err := conn.Write(handshakeMessage.Bytes())
	if err != nil {
		return nil, createError("handleHandshake()", err.Error())
	} else {
		printWithColor(Yellow, fmt.Sprint(" Hanshake Bytes sent:", n))
	}

	data := make([]byte, 2048)
	n, err = conn.Read(data)
	if err != nil {
		return nil, createError("handleHandshake()", err.Error())
	}
	printWithColor(Green, "Hanshake succesful")
	return data, nil
}

// Requests a block of a piece, normaly a piece is formed by various blocks
func requestBlock(conn net.Conn, fileIndex int, blockOffset int, blockLength int) ([]byte, error) {

	//printWithColor(Red, fmt.Sprint("requesting index: ", fileIndex, " block offset: ", blockOffset))

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

	//clean up the connection if there is anything there yet
	var response = make([]byte, 40000)
	totalRead := 0
	actualData := []byte{}
	for {
		conn.SetReadDeadline(time.Now().Add(5000 * time.Millisecond))
		n, err = conn.Read(response)
		totalRead += n

		if n == 5 {
			fmt.Println("this is shit data that does not serve for anything", response[:n])
			continue
		}

		actualData = append(actualData, response[:n]...)

		if err != nil {
			if totalRead > 13 {
				return actualData[13:], nil
			}
			//if there is an error but we haven't tried twice yet, we try again
			return nil, createError("requestBlock() on conn.Write()", err.Error())
		}

		if totalRead > 15000 {
			return actualData[13:], nil
		}

		//length := binary.BigEndian.Uint32(block[:4])
		//index := binary.BigEndian.Uint32(block[6:10])
		//begin := binary.BigEndian.Uint32(block[10:14])

		//sometimes the connections sends just 5 random bytes instead of the actual data, so we just keep looping if that happens
		//if len(block[:n]) <= 10 {
		//	printWithColor(Yellow, " THIS SHIT IS NOT the data ")
		//}

	}

	//gottenFile := response[13:n]
	//start := fileIndex * 131072
	//expectedFile := GetExpectedFile()[start+blockOffset : start+blockOffset+blockLength]
	//filesDoMatch := reflect.DeepEqual(gottenFile, expectedFile)
	//printWithColor(Yellow, fmt.Sprint("Match? ", filesDoMatch))
	//return only the data, not the metadata

	//return block[13:totalRead], nil

}

func listenToIncomingData(port int) {
	fmt.Println("listening TO data on port ", port)
	l, err := net.Listen("tcp", fmt.Sprint(":", port))
	if err != nil {
		log.Println(err)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go handleConnection(conn)
	}
}
func handleConnection(conn net.Conn) {
	var read = make([]byte, 30000)

	n, err := conn.Read(read)
	fmt.Println("Data read")
	if err != nil {
		printWithColor(Red, fmt.Sprint("error on reading connection on handleConnection()", err.Error()))
		return
	}

	if n == 0 {
		log.Println("Finished this shit lmao")
	}
	printWithColor(Green, fmt.Sprint("Data gotten: ", read[:n]))

}
