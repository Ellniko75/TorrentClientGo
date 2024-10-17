package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	"github.com/jackpal/bencode-go"
)

var Red = "\033[31m"
var Green = "\033[32m"
var Yellow = "\033[33m"
var Blue = "\033[34m"
var Magenta = "\033[35m"
var Cyan = "\033[36m"
var Gray = "\033[37m"
var White = "\033[97m"

func main() {
	file, err := os.Open("./db.torrent")
	defer file.Close()
	if err != nil {
		log.Println("error on reading file")
	}

	var torrent TorrentFile
	err = bencode.Unmarshal(file, &torrent)
	if err != nil {
		log.Println("Error on unmarshaling")
	}

	TorrentFileToBuild := TorrentFileToBuild{}
	loadHashes(&torrent, &TorrentFileToBuild)
	loadTrackers(&torrent, &TorrentFileToBuild)
	requestPieces(&TorrentFileToBuild)
}

func generatePeerID() ([20]byte, error) {

	var peerId bytes.Buffer

	firstPart := []byte("-GO0001-")
	restOfTheString := []byte(randomString(12))

	if err := binary.Write(&peerId, binary.BigEndian, firstPart); err != nil {
		log.Println(err)
	}
	if err := binary.Write(&peerId, binary.BigEndian, restOfTheString); err != nil {
		log.Println(err)
	}

	if len(peerId.Bytes()) != 20 {
		return [20]byte(peerId.Bytes()), createError("generatePeerId()", " The peer ID is not 20 bytes long")
	}

	return [20]byte(peerId.Bytes()), nil
}

// Function to create a random string (for peer ID)
func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	s := make([]rune, n)
	for i := range s {
		s[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(s)
}

func requestPieces(torrentToBuild *TorrentFileToBuild) {
	for _, hash := range torrentToBuild.ListOfHashes {
		for _, tracker := range torrentToBuild.ListOfTrackers {

			if tracker[:3] == "udp" {
				//Adjust the format of the UDP tracker URL
				trackerURL := strings.TrimPrefix(tracker, "udp://") //you need to strip the udp:// from the tracker to resolve the address later
				trackerURL = strings.TrimSuffix(trackerURL, "/announce")

				//create udp connection for the UDP tracker
				conn, err := createUdpConnection(trackerURL)
				if err != nil {
					printWithColor(Red, err.Error())
					continue
				}

				//Add a timeout for the connection
				timeoutDuration := 3 * time.Second
				conn.SetDeadline(time.Now().Add(timeoutDuration))

				//Create random transaction ID
				transactionID := int32(rand.Int31())
				//Request to UDP TRACKER and read the response
				transactionIDResponse, connectionIDResponse, err := initiateUdpConnection(conn, transactionID)
				if err != nil {
					printWithColor(Red, err.Error())
					continue
				}

				//GET THE PEERS THAT HAVE THE FILE
				peerID, err := generatePeerID()
				if err != nil {
					printWithColor(Red, err.Error())
					continue
				}

				trackerAnnounceResponse, bytesRead, err := scrapeIpsFromTracker(
					conn,
					hash,
					connectionIDResponse,
					transactionIDResponse,
					peerID)

				if err != nil {
					printWithColor(Red, err.Error())
					continue
				}

				//Get all the information from the tracker response
				responseAction := binary.BigEndian.Uint32(trackerAnnounceResponse[:4])
				responseTransaction := binary.BigEndian.Uint32(trackerAnnounceResponse[4:8])
				interval := binary.BigEndian.Uint32(trackerAnnounceResponse[8:12])
				leechers := binary.BigEndian.Uint32(trackerAnnounceResponse[12:16])
				seeders := binary.BigEndian.Uint32(trackerAnnounceResponse[16:20])
				printWithColor(Blue, "---------------------------------")
				printWithColor(Green, fmt.Sprint("Current tracker > ", trackerURL))
				printWithColor(Green, fmt.Sprint("Bytes totales >", bytesRead))
				printWithColor(Green, fmt.Sprint("Action> ", responseAction))
				printWithColor(Green, fmt.Sprint("transaction> ", responseTransaction))
				printWithColor(Green, fmt.Sprint("Interval> ", interval))
				printWithColor(Green, fmt.Sprint("Leechers> ", leechers))
				printWithColor(Green, fmt.Sprint("Seeders> ", seeders))
				printWithColor(Blue, "---------------------------------")
				fmt.Println("IPS > ", binary.BigEndian.Uint32(trackerAnnounceResponse[20:]))

				conn.Close()

			}

		}
		/*
			hashUrlEncoded := url.QueryEscape(string(hash))
			peerID := generatePeerID()

			params := url.Values{
				"info_hash":  {hashUrlEncoded},
				"peer_id":    {peerID},
				"port":       {"6881"},    // Example port
				"uploaded":   {"0"},       // Total bytes uploaded (initially 0)
				"downloaded": {"0"},       // Total bytes downloaded (initially 0)
				"left":       {"1000000"}, // Total bytes left to download (set appropriately)
				"compact":    {"1"},       // Compact peer list (1 for compact, 0 for non-compact)
				"event":      {"started"}, // Event can be "started", "stopped", "completed"
			}*/

	}
}

func loadHashes(torrentInfo *TorrentFile, toBuild *TorrentFileToBuild) {
	pieceHashes := []byte(torrentInfo.Info.Pieces)
	numPieces := len(pieceHashes) / 20 // Each piece hash is 20 bytes (SHA1)

	for i := 0; i < numPieces; i++ {
		hash := [20]byte(pieceHashes[i*20 : (i+1)*20])

		toBuild.ListOfHashes = append(toBuild.ListOfHashes, hash)
	}
}
func loadTrackers(torrentInfo *TorrentFile, toBuild *TorrentFileToBuild) {
	for _, tracker := range torrentInfo.AnnounceList {
		trackerToStr := strings.Join(tracker, "")
		toBuild.ListOfTrackers = append(toBuild.ListOfTrackers, trackerToStr)
	}
}

// Returns TransactionID, ConnectionID, error
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

	fmt.Println("Connect request sent.")

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

func scrapeIpsFromTracker(conn *net.UDPConn, hash [20]byte, connectionId uint64, transactionID uint32, peerID [20]byte) ([]byte, int, error) {
	currentFunctionName := "scrapeIpsFromTracker()"

	if len(peerID) != 20 {
		log.Println("PEER ID MUST BE OF 20 BYTES")
	}
	if len(hash) != 20 {
		log.Println("hash is not 20 bytes long")
	}

	//CREATE THE PACKET TO SEND
	var packet bytes.Buffer

	//PACKET FORMAT
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
	if err := binary.Write(&packet, binary.BigEndian, int64(0)); err != nil {
		log.Fatal("Error writing downloaded:", err)
	}
	if err := binary.Write(&packet, binary.BigEndian, int64(0)); err != nil {
		log.Fatal("Error writing left:", err)
	}
	if err := binary.Write(&packet, binary.BigEndian, int64(0)); err != nil {
		log.Fatal("Error writing uploaded:", err)
	}
	if err := binary.Write(&packet, binary.BigEndian, int32(0)); err != nil {
		log.Fatal("Error writing event:", err)
	}
	if err := binary.Write(&packet, binary.BigEndian, int32(0)); err != nil {
		log.Fatal("Error writing IP address:", err)
	}
	if err := binary.Write(&packet, binary.BigEndian, int32(44315690)); err != nil {
		log.Fatal("Error writing key:", err)
	}
	if err := binary.Write(&packet, binary.BigEndian, uint32(0xFFFFFFFF)); err != nil { // request unlimited peers
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

	trackerAnnounceResponse := make([]byte, 1024)
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
