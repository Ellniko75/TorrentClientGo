package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
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

func generatePeerID() string {
	peerID := "-GO0001-" + randomString(12)
	return peerID
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
				trackerURL := strings.TrimPrefix(tracker, "udp://") //you need to strip the udp:// from the tracker to resolve the address later
				trackerURL = strings.TrimSuffix(trackerURL, "/announce")
				transactionID := int32(12345)

				// Resolve UDP address
				addr, err := net.ResolveUDPAddr("udp", trackerURL)
				if err != nil {
					fmt.Println("Error resolving address:", err)
					return
				}
				// Create UDP connection
				conn, err := net.DialUDP("udp", nil, addr)
				if err != nil {
					fmt.Println("Error creating UDP connection:", err)
					return
				}
				//Add a timeout for the connection
				timeoutDuration := 10 * time.Second
				conn.SetDeadline(time.Now().Add(timeoutDuration))

				//Request to UDP TRACKER and read the response
				initiateUdpConnection(conn, transactionID)
				response := make([]byte, 16)
				n, _, err := conn.ReadFromUDP(response)
				if err != nil {
					log.Println("ERROR ON TRYING TO CONNECT TO UDP TRACKER")
				}
				//Get the transaction ID and connectionID from the response
				transactionIDResponse := binary.BigEndian.Uint32(response[4:8])
				connectionIDResponse := binary.BigEndian.Uint64(response[8:16])

				scrapeIpsFromTracker(conn, hash, connectionIDResponse, transactionIDResponse, generatePeerID())
				trackerAnnounceResponse := make([]byte, 1024)
				n, _, err = conn.ReadFrom(trackerAnnounceResponse)

				if err != nil {
					log.Println("ERROR ON READING ANNOUNCE RESPONSE")
				}
				responseAction := binary.BigEndian.Uint32(trackerAnnounceResponse[:4])
				responseTransaction := binary.BigEndian.Uint32(trackerAnnounceResponse[4:8])
				interval := binary.BigEndian.Uint32(trackerAnnounceResponse[8:12])
				leechers := binary.BigEndian.Uint32(trackerAnnounceResponse[12:16])
				seeders := binary.BigEndian.Uint32(trackerAnnounceResponse[16:20])

				if responseAction == 3 {
					toStr := string(trackerAnnounceResponse[8:])
					printWithColor(Red, fmt.Sprint("ERROR: ", toStr))
					printWithColor(Blue, "---------------------------------")
					continue
				}

				printWithColor(Blue, "---------------------------------")
				printWithColor(Green, fmt.Sprint("Current tracker: ", trackerURL))
				printWithColor(Green, fmt.Sprint("Bytes totales: ", n))
				printWithColor(Green, fmt.Sprint("Action> ", responseAction))
				printWithColor(Green, fmt.Sprint("transaction> ", responseTransaction))
				printWithColor(Green, fmt.Sprint("Interval> ", interval))
				printWithColor(Green, fmt.Sprint("Leechers> ", leechers))
				printWithColor(Green, fmt.Sprint("Seeders> ", seeders))
				printWithColor(Blue, "---------------------------------")

				//fmt.Println("IPS > ", binary.BigEndian.Uint32(trackerAnnounceResponse[20:]))

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
		hash := pieceHashes[i*20 : (i+1)*20]
		toBuild.ListOfHashes = append(toBuild.ListOfHashes, hash)
	}
}
func loadTrackers(torrentInfo *TorrentFile, toBuild *TorrentFileToBuild) {
	for _, tracker := range torrentInfo.AnnounceList {
		trackerToStr := strings.Join(tracker, "")
		toBuild.ListOfTrackers = append(toBuild.ListOfTrackers, trackerToStr)
	}
}

// this returns the connectionID and transactionID so that we can start getting the IPs of people who have the file
func initiateUdpConnection(conn *net.UDPConn, transactionID int32) {
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
		fmt.Println("Error sending request:", err)
		return
	}
	fmt.Println("Connect request sent.")
}
func scrapeIpsFromTracker(conn *net.UDPConn, hash []byte, connectionId uint64, transactionID uint32, peerID string) {
	if len(peerID) != 20 {
		log.Println("PEER ID MUST BE OF 20 BYTES")
	}
	if len(hash) != 20 {
		log.Println("hash is not 20 bytes long")
	}
	var buf bytes.Buffer

	//connectionID 8 bytes
	binary.Write(&buf, binary.BigEndian, connectionId)
	//action 4 bytes
	binary.Write(&buf, binary.BigEndian, uint32(1))
	//transaction id 4 bytes
	binary.Write(&buf, binary.BigEndian, transactionID)
	//hash piece 20 bytes
	binary.Write(&buf, binary.BigEndian, hash)
	//Peer ID 20 bytes
	binary.Write(&buf, binary.BigEndian, peerID)
	//Downloaded 8 bytes
	binary.Write(&buf, binary.BigEndian, uint64(0))
	//Left 8 bytes
	binary.Write(&buf, binary.BigEndian, uint64(0))
	//Uploaded 8 bytes
	binary.Write(&buf, binary.BigEndian, uint64(0))
	//Event 4 bytes
	binary.Write(&buf, binary.BigEndian, uint32(0))
	//IP address 4 bytes
	binary.Write(&buf, binary.BigEndian, uint32(0))
	//key 4 bytes
	binary.Write(&buf, binary.BigEndian, uint32(44315690))
	//Number of peers you want 4 bytes
	binary.Write(&buf, binary.BigEndian, int32(-1))
	//port for listening to peer connections 2 bytes
	binary.Write(&buf, binary.BigEndian, uint16(6881))

	// Send the request
	_, err := conn.Write(buf.Bytes())
	if err != nil {
		fmt.Println("Error sending Announce request:", err)
		return
	}
	fmt.Println("Anounce Sent")
}
