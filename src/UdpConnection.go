package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"time"
)

// Creates udp connection with a timeout of 2 seconds
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

	//Add a timeout for the connection
	timeoutDuration := 2 * time.Second
	conn.SetDeadline(time.Now().Add(timeoutDuration))
	return conn, nil
}

// In charge of initiating the connection with the tracker, returns the connectionID needed for requesting peers. Returns TransactionID, ConnectionID, error.
func initiateUdpConnection(conn *net.UDPConn, transactionID int32) (uint32, uint64, error) {
	currentFunctionName := "initiateUdpConnection()"

	// Construct a connect request (binary format)
	var buf bytes.Buffer
	// Magic constant for connect request: 0x41727101980
	binary.Write(&buf, binary.BigEndian, int64(0x41727101980))
	// Action: 0 (for connect request)
	binary.Write(&buf, binary.BigEndian, int32(0))
	// Transaction ID: Random (e.g., 12345)
	binary.Write(&buf, binary.BigEndian, transactionID)

	// Send the request
	_, err := conn.Write(buf.Bytes())
	if err != nil {
		return 0, 0, createError(currentFunctionName, err.Error())
	}

	//Read Response
	response := make([]byte, 16)
	_, _, err = conn.ReadFromUDP(response)
	if err != nil {
		return 0, 0, createError(currentFunctionName, err.Error())
	}
	//Get the transaction ID and connectionID from the response
	transactionIDResponse := binary.BigEndian.Uint32(response[4:8])
	connectionIDResponse := binary.BigEndian.Uint64(response[8:])

	//Assure that both transactionIDs are the same
	if transactionID != int32(transactionIDResponse) {
		return transactionIDResponse, connectionIDResponse, createError(currentFunctionName, " TransactionID is not the same as the TransactionIDResponse")
	}

	return transactionIDResponse, connectionIDResponse, nil
}

func getEventTypeForUDPTracker(firstTime bool, torrentBuilding *TorrentFileToBuild) int32 {
	if firstTime {
		return 1
	}
	isFinished := torrentBuilding.TotalDownloaded.Downloaded == int64(torrentBuilding.FileLength)
	if isFinished {
		return 3
	}

	return 0
}

func getPeersFromUdp(conn *net.UDPConn, hash []byte, connectionId uint64, transactionID uint32, peerID [20]byte, downloaded int64, left int64, uploaded int64, event int32) ([]byte, int, error) {
	currentFunctionName := "getPeers()"

	//CREATE THE PACKET TO SEND
	var packet bytes.Buffer

	//PACKET FORMAT
	//64-bit integer	connection_id
	//32-bit integer	action	1
	//32-bit integer	transaction_id
	//20-byte string	info_hash
	//20-byte string	peer_id
	//64-bit integer	downloaded
	//64-bit integer	left
	//64-bit integer	uploaded
	//32-bit integer	event
	//32-bit integer	IP address	0
	//32-bit integer	key
	//32-bit integer	num_want	-1
	//16-bit integer	port

	if err := binary.Write(&packet, binary.BigEndian, connectionId); err != nil {
		log.Fatal("Error writing connectionId:", err)
	}
	if err := binary.Write(&packet, binary.BigEndian, uint32(1)); err != nil {
		log.Fatal("Error writing action:", err)
	}
	if err := binary.Write(&packet, binary.BigEndian, transactionID); err != nil {
		log.Fatal("Error writing transactionID:", err)
	}
	if err := binary.Write(&packet, binary.BigEndian, hash); err != nil {
		log.Fatal("Error writing hash:", err)
	}
	if err := binary.Write(&packet, binary.BigEndian, peerID); err != nil {
		log.Fatal("Error writing peerID:", err)
	}
	if err := binary.Write(&packet, binary.BigEndian, int64(downloaded)); err != nil {
		log.Fatal("Error writing downloaded:", err)
	}
	//this is the left parameter, somehow if i put 0 here, it will return my own IP and port from the qbitorrent that has the file downloaded
	//but if i put this to the actual size of data that's left to download, it will only return my own ip with the port THIS program is using
	//idk
	if err := binary.Write(&packet, binary.BigEndian, int64(0)); err != nil {
		log.Fatal("Error writing left:", err)
	}
	if err := binary.Write(&packet, binary.BigEndian, int64(uploaded)); err != nil {
		log.Fatal("Error writing uploaded:", err)
	}
	if err := binary.Write(&packet, binary.BigEndian, int32(event)); err != nil {
		log.Fatal("Error writing event:", err)
	}
	if err := binary.Write(&packet, binary.BigEndian, int32(0)); err != nil {
		log.Fatal("Error writing IP address:", err)
	}
	if err := binary.Write(&packet, binary.BigEndian, int32(64375677)); err != nil {
		log.Fatal("Error writing key:", err)
	}
	if err := binary.Write(&packet, binary.BigEndian, int32(-1)); err != nil { // request unlimited peers
		log.Fatal("Error writing number of peers:", err)
	}
	if err := binary.Write(&packet, binary.BigEndian, int16(6881)); err != nil {
		log.Fatal("Error writing port:", err)
	}

	// Send the request
	_, err := conn.Write(packet.Bytes())
	if err != nil {
		return nil, 0, createError(currentFunctionName, err.Error())
	}
	fmt.Println("Anounce Sent")

	trackerAnnounceResponse := make([]byte, 50000)
	bytesRead, _, err := conn.ReadFrom(trackerAnnounceResponse)
	if err != nil {
		return nil, bytesRead, createError(currentFunctionName, err.Error())
	}

	//check if the response action is 3, if that's the case then there was an error and the udp tracker is letting us know
	responseAction := binary.BigEndian.Uint32(trackerAnnounceResponse[:4])
	if responseAction == 3 {
		errorString := string(trackerAnnounceResponse[8:])
		return trackerAnnounceResponse, bytesRead, errors.New(fmt.Sprint("Error on scrapeIpsFromTracker(): ", errorString))
	}

	return trackerAnnounceResponse, bytesRead, nil

}
