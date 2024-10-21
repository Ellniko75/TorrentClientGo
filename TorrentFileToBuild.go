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
	ipsWithTheFile []string
	FilesLength    int
	MainTracker    string
	ListOfTrackers []string //list of all the trackers
	ListOfHashes   [][]byte //hashes for each piece of the file
	InfoHash       []byte
	File           []byte //property to write the file when the pieces arrive
}

// TODO: for now it loads the infohash forcefully, because this library is complete shit and cannot for the life of it calculate the infohash
func (this *TorrentFileToBuild) loadHashes(torrentInfo *TorrentFileInfo) {

	hashLen := 20 //sha1 length
	for i := 0; i < len(torrentInfo.Info.Pieces); i += hashLen {
		currentHash := torrentInfo.Info.Pieces[i : i+hashLen]
		this.ListOfHashes = append(this.ListOfHashes, []byte(currentHash))
	}
}
func (this *TorrentFileToBuild) loadInfoHash(hash []byte) {
	this.InfoHash = hash
}
func (this *TorrentFileToBuild) loadTrackers(torrentInfo *TorrentFileInfo) {
	this.MainTracker = torrentInfo.Announce
	for _, tracker := range torrentInfo.AnnounceList {
		trackerToStr := strings.Join(tracker, "")
		this.ListOfTrackers = append(this.ListOfTrackers, trackerToStr)
	}
}
func (this *TorrentFileToBuild) getPeers() {
	//for {
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
				this.InfoHash,
				connectionIDResponse,
				transactionIDResponse,
				peerID)
			if err != nil {
				printWithColor(Red, err.Error())
				continue
			}

			//parse the tracker response
			TrackerResponseParsed := TrackerResponse{}
			TrackerResponseParsed.Create(trackerAnnounceResponse)
			TrackerResponseParsed.Print()
			ipsAndPorts := TrackerResponseParsed.getIpAndPorts()

			for _, v := range ipsAndPorts {
				printWithColor(Red, v)
				this.ipsWithTheFile = append(this.ipsWithTheFile, v)
			}

			//CLOSE THE CONNECTION
			conn.Close()
		}

	}
	//time.Sleep(10 * time.Second)
	//}
}

func (this *TorrentFileToBuild) downloadFile() {
	for fileIndex, hash := range this.ListOfHashes {
		this.askPeersForFile(fileIndex, hash)
	}
}
func (this *TorrentFileToBuild) askPeersForFile(fileIndex int, hash []byte) ([]byte, error) {
	//request the file to the peers
	for _, ip := range this.ipsWithTheFile {
		fmt.Printf("DONT PAY ATTENTION TO THIS: %v\n", ip)
		peerID, err := generatePeerID()
		if err != nil {
			log.Print("Error generating peerID", err)
		}
		err = connectToPeerAndRequestFile(ip, fileIndex, hash, peerID)
		if err != nil {
			log.Println(err)
		}
	}

	return nil, nil
}
