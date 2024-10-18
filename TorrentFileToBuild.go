package main

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"
)

// contains all the info and functions necessary to download the file
type TorrentFileToBuild struct {
	FilesLength    int
	MainTracker    string
	ListOfTrackers []string   //list of all the trackers
	ListOfHashes   [][20]byte //hashes for each piece of the file
	File           []byte     //property to write the file when the pieces arrive
}

func (this *TorrentFileToBuild) loadHashes(torrentInfo *TorrentFileInfo) {
	this.FilesLength = torrentInfo.Info.PieceLength

	pieceHashes := []byte(torrentInfo.Info.Pieces)
	numPieces := len(pieceHashes) / 20 // Each piece hash is 20 bytes (SHA1)
	for i := 0; i < numPieces; i++ {
		hash := [20]byte(pieceHashes[i*20 : (i+1)*20])

		this.ListOfHashes = append(this.ListOfHashes, hash)
	}
}
func (this *TorrentFileToBuild) loadTrackers(torrentInfo *TorrentFileInfo) {
	this.MainTracker = torrentInfo.Announce
	for _, tracker := range torrentInfo.AnnounceList {
		trackerToStr := strings.Join(tracker, "")
		this.ListOfTrackers = append(this.ListOfTrackers, trackerToStr)
	}
}
func (this *TorrentFileToBuild) downloadPieces() {
	for fileIndex, hash := range this.ListOfHashes {
		for _, tracker := range this.ListOfTrackers {
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

				//GENERATE A RANDOM ID FOR THE REQUEST
				peerID, err := generatePeerID()
				if err != nil {
					printWithColor(Red, err.Error())
					continue
				}
				//GET ALL THE PEERS THAT HAVE THE FILE
				trackerAnnounceResponse, _, err := scrapeIpsFromTracker(
					conn,
					hash,
					connectionIDResponse,
					transactionIDResponse,
					peerID)
				if err != nil {
					printWithColor(Red, err.Error())
					continue
				}
				TrackerResponseParsed := TrackerResponse{}
				TrackerResponseParsed.Create(trackerAnnounceResponse)
				TrackerResponseParsed.Print()

				ipsAndPorts := trackerAnnounceResponse[20:]

				//loop the seeders and request the file
				for i := 0; i < int(TrackerResponseParsed.Seeders)*6; i = i + 6 {
					first := ipsAndPorts[i]
					second := ipsAndPorts[i+1]
					third := ipsAndPorts[i+2]
					fourth := ipsAndPorts[i+3]
					portPart1 := ipsAndPorts[i+4]
					portPart2 := ipsAndPorts[i+5]
					ip := fmt.Sprint(first, ".", second, ".", third, ".", fourth, ":", portPart1, "", portPart2)

					data, err := connectToPeerAndRequestFile(ip, fileIndex, hash, peerID)
					if err != nil {
						log.Println(err)
					} else {
						fmt.Println("GOT THE DATA: ", data)
					}

				}
				//CLOSE THE CONNECTION
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