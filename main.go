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

type TorrentFile struct {
	Announce string `bencode:"announce"`
	Info     struct {
		Pieces      string `bencode:"pieces"`
		PieceLength int    `bencode:"piece length"`
	} `bencode:"info"`
	AnnounceList [][]string `bencode:"announce-list"` // Optional multiple trackers
}

type TorrentFileToBuild struct {
	ListOfTrackers []string //list of all the trackers
	ListOfHashes   [][]byte //hashes for each piece of the file
	File           []byte   //property to write the file when the pieces arrive
}

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
				fmt.Println(hash[0])
				trackerURL := strings.TrimPrefix(tracker, "udp://") //you need to strip the udp:// from the tracker to resolve the address later
				transactionID := int32(12345)

				// Resolve UDP address
				addr, err := net.ResolveUDPAddr("udp", trackerURL)
				if err != nil {
					fmt.Println("Error resolving address:", err)
					return
				}
				// Dial UDP connection
				conn, err := net.DialUDP("udp", nil, addr)
				if err != nil {
					fmt.Println("Error creating UDP connection:", err)
					return
				}
				timeoutDuration := 1 * time.Second
				conn.SetDeadline(time.Now().Add(timeoutDuration))

				requestToUDPTracker(conn, transactionID)
				response := make([]byte, 16)
				conn.ReadFromUDP(response)

				transactionIDResponse := binary.BigEndian.Uint32(response[4:8])
				connectionIDResponse := binary.BigEndian.Uint32(response[8:16])

				fmt.Println("Transaction ID: ", transactionIDResponse, " Connection ID: ", connectionIDResponse)

				fmt.Println(response)
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

func requestToUDPTracker(conn *net.UDPConn, transactionID int32) {
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
