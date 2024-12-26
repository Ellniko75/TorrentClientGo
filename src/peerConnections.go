package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

// Connects to the peer anr requests the whole file, block by block
func connectToPeerAndRequestWholePiece(conn *Connection, fileIndex int, torrentInfo *TorrentFileToBuild, final bool) ([]byte, error) {
	AmountOfBlocks := torrentInfo.AmountOfBlocks
	BlockLength := torrentInfo.BlockLength
	//if we are downloading the final piece, we have to adjust the block length, since it usually happens that you cannot divide all files equally by 16kb
	if final {
		//we have to download the whole thing. but we have downloaded everything but the last piece
		haveToDownload := torrentInfo.FileLength
		//we calculate all the bytes downloaded up until the last piece (we use totalPieces and not totalPieces-1 because its an index and they start at index 0)
		piecesSummedCalculation := torrentInfo.TotalPieces * torrentInfo.PieceSize
		missing := haveToDownload - piecesSummedCalculation
		divisions := getDivisibleNumber(missing)
		AmountOfBlocks = divisions
		BlockLength = missing / AmountOfBlocks

		fmt.Println("Missing: ", missing, "block length: ", BlockLength, "Amount of blocks: ", AmountOfBlocks)
	}

	wholePiece := []byte{}
	for i := 0; i < AmountOfBlocks; i++ {
		blockOffset := BlockLength * i
		//send the request for the data
		data, err := requestBlock(conn.Conn, fileIndex, blockOffset, BlockLength)
		if err != nil {
			return nil, err
		}
		//time.Sleep(1 * time.Second)
		if len(data) > 0 {
			wholePiece = append(wholePiece, data...)
		}
	}

	//for all the other pieces the peers do not send that 5 bytes, so que return the whole piece
	return wholePiece, nil
}
func initiatePeerConnection(ip string, infoHash []byte, peerId [20]byte) (net.Conn, map[int]bool, error) {
	//create the connection based on the IP
	connection, err := createTcpConnection(ip)
	if err != nil {
		return nil, nil, err
	}

	_, bitfield, err := handleHandshake(infoHash, peerId, connection)
	if err != nil {
		return nil, bitfield, err
	}

	return connection, bitfield, nil
}

// Creates the tcp connection and dials up with the url, for now it's hardcoded to request to the port I know its opened, since I cannot make the port be good
func createTcpConnection(ip string) (net.Conn, error) {
	// Connect to the server
	//printWithColor(Yellow, fmt.Sprint(" Attempting to Connect: ", ip))
	conn, err := net.DialTimeout("tcp", ip, 3*time.Second)
	if err != nil {
		return nil, createError("createTcpConnection()", err.Error())
	}
	//printWithColor(Yellow, fmt.Sprint(" Connected to: ", ip))
	return conn, nil
}

func handleHandshake(infoHash []byte, peerID [20]byte, conn net.Conn) ([]byte, map[int]bool, error) {

	//handle the handshake
	var handshakeMessage bytes.Buffer

	//Write the length (pstrlen)
	if err := binary.Write(&handshakeMessage, binary.BigEndian, byte(19)); err != nil {
		return nil, nil, createError("handleHandshake()", err.Error())
	}
	//Write the protocol (pstr)
	if err := binary.Write(&handshakeMessage, binary.BigEndian, []byte("BitTorrent protocol")); err != nil {
		return nil, nil, createError("handleHandshake()", err.Error())
	}
	//Write the reserved 8 bytes (reserved)
	if err := binary.Write(&handshakeMessage, binary.BigEndian, make([]byte, 8)); err != nil {
		return nil, nil, createError("handleHandshake()", err.Error())
	}
	//Write the hash (info_hash)
	if err := binary.Write(&handshakeMessage, binary.BigEndian, infoHash); err != nil {
		return nil, nil, createError("handleHandshake()", err.Error())
	}
	//Write the peerID (peer_id)
	if err := binary.Write(&handshakeMessage, binary.BigEndian, peerID); err != nil {
		return nil, nil, createError("handleHandshake()", err.Error())
	}
	//send the handshake message
	_, err := conn.Write(handshakeMessage.Bytes())
	if err != nil {
		return nil, nil, createError("handleHandshake()", err.Error())
	}
	//set the deadline for handhsake to 3 seconds
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	data := make([]byte, 2048)
	_, err = conn.Read(data)
	if err != nil {
		return nil, nil, createError("handleHandshake()", err.Error())
	}

	lengthOfProtocolmsg := byte(data[:1][0])
	//messageProtocol := data[1 : 1+lengthOfProtocolmsg]
	start := 1 + lengthOfProtocolmsg
	//reservedResp := data[start : start+8]
	start2 := start + 8
	//infoHashResp := data[start2 : start2+20]
	start3 := start2 + 20
	//peerIdResp := data[start3 : start3+20]
	start4 := start3 + 20
	bitfieldResponse := data[start4:]
	bitfieldResponseLength := binary.BigEndian.Uint32(bitfieldResponse[:4])
	fullData := bitfieldResponse[5:(5 + bitfieldResponseLength)]
	printWithColor(Green, fmt.Sprint("Hanshake succesful, BITFIELD: ", fullData))

	BitfieldParsed := createBitfieldMap(fullData)

	//reset deadline
	conn.SetReadDeadline(time.Time{})
	return data, BitfieldParsed, nil
}

func createBitfieldMap(bitfieldNetworkResponse []byte) map[int]bool {
	BitfieldMap := map[int]bool{}
	index := 0
	for _, v := range bitfieldNetworkResponse {
		binaryInString := fmt.Sprintf("%08b", v)
		BitfieldMap[index] = oneToTrueAndZeroToFalse(binaryInString[:1])
		index++
		BitfieldMap[index] = oneToTrueAndZeroToFalse(binaryInString[1:2])
		index++
		BitfieldMap[index] = oneToTrueAndZeroToFalse(binaryInString[2:3])
		index++
		BitfieldMap[index] = oneToTrueAndZeroToFalse(binaryInString[3:4])
		index++
		BitfieldMap[index] = oneToTrueAndZeroToFalse(binaryInString[4:5])
		index++
		BitfieldMap[index] = oneToTrueAndZeroToFalse(binaryInString[5:6])
		index++
		BitfieldMap[index] = oneToTrueAndZeroToFalse(binaryInString[6:7])
		index++
		BitfieldMap[index] = oneToTrueAndZeroToFalse(binaryInString[7:8])
		index++
	}
	return BitfieldMap
}
func oneToTrueAndZeroToFalse(i string) bool {
	if i == "0" {
		return false
	}
	return true
}

// Requests a block of a piece, normaly a piece is formed by various blocks
func requestBlock(conn net.Conn, fileIndex int, blockOffset int, blockLength int) ([]byte, error) {
	err := sendInterestedPayloadToConnection(conn)
	if err != nil {
		return nil, err
	}

	actualData := []byte{}
	var response = make([]byte, 100000)
	for {
		//Read the connection data and store it on response
		n, err := conn.Read(response)
		if err != nil {
			//if there is an error but we haven't tried twice yet, we try again
			return nil, createError("requestBlock() on conn.Write()", fmt.Sprint(err.Error(), " on ip: ", conn.RemoteAddr()))
		}

		//get the type of message we got
		idOfMessage := getIdOfPeerMessage(response[:n])

		//handle the metadata response
		responseIsMetaData := n == 5
		if responseIsMetaData {
			//if it is an unchoke message we send the request
			if idOfMessage == 1 {
				err = sendRequestPayloadToConnection(conn, fileIndex, blockOffset, blockLength)
				if err != nil {
					return nil, err
				}
			}
			if idOfMessage == 0 {
				fmt.Println("I GOT CHOKED BY: ", conn.RemoteAddr())
			}
			//we always continue listening for data if we encounter metadata since we don't want to append it to the actual data of the file
			continue
		}

		//add the data
		actualData = append(actualData, response[:n]...)

		//if we got enough data we return it
		if len(actualData) >= blockLength {
			return actualData[13:], nil
		}
	}
}
func sendRequestPayloadToConnection(conn net.Conn, fileIndex int, blockOffset int, blockLength int) error {
	//load the payload to send to the peer
	var buff bytes.Buffer
	//Size of the request (Message Length)
	if err := binary.Write(&buff, binary.BigEndian, int32(13)); err != nil {
		return createError("sendRequestPayloadToConnection() Message Length ", err.Error())
	}
	//indicate that this is a request with the 6 (Message ID)
	if err := binary.Write(&buff, binary.BigEndian, byte(6)); err != nil {
		return createError("sendRequestPayloadToConnection() Message ID  ", err.Error())
	}
	//The index of the piece being requested. (Piece Index)
	if err := binary.Write(&buff, binary.BigEndian, int32(fileIndex)); err != nil {
		return createError("sendRequestPayloadToConnection() Piece Index", err.Error())
	}
	//Block Offset
	if err := binary.Write(&buff, binary.BigEndian, int32(blockOffset)); err != nil {
		return createError("sendRequestPayloadToConnection() Block Length", err.Error())
	}
	//Block length
	if err := binary.Write(&buff, binary.BigEndian, int32(blockLength)); err != nil {
		return createError("sendRequestPayloadToConnection() ", err.Error())
	}

	//send the payload requesting the file
	_, err := conn.Write(buff.Bytes())

	return err
}
func sendInterestedPayloadToConnection(conn net.Conn) error {
	var buff bytes.Buffer
	//Write Response Length
	if err := binary.Write(&buff, binary.BigEndian, int32(1)); err != nil {
		return createError("sendRequestPayloadToConnection() ", err.Error())
	}
	if err := binary.Write(&buff, binary.BigEndian, byte(2)); err != nil {
		return createError("sendRequestPayloadToConnection() ", err.Error())
	}
	//send the payload requesting the file
	_, err := conn.Write(buff.Bytes())

	return err
}

func getIdOfPeerMessage(data []byte) byte {
	idOfMessage := data[4:5]
	return idOfMessage[0]
}
