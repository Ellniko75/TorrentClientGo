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
	PieceSize      int //size of each piece to download
	TotalPieces    int
	BlockLength    int
	AmountOfBlocks int
	ipsWithTheFile []string //slice of all peers that have the file
	MainTracker    string
	ListOfTrackers []string //list of all the trackers
	ListOfHashes   [][]byte //hashes for each piece of the file
	InfoHash       []byte
	FileLength     int
	File           [][]byte //property to write the file when the pieces arrive
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
func (this *TorrentFileToBuild) CalculateTotalPiecesAndBlockLength(info TorrentFileInfo) {
	this.FileLength = info.Info.Length
	this.PieceSize = info.Info.PieceLength
	this.TotalPieces = this.FileLength / this.PieceSize
	this.BlockLength = 16384
	this.AmountOfBlocks = this.PieceSize / this.BlockLength //Calculate the amount of blocks per piece

	printWithColor(Red, fmt.Sprint("FILE TOTAL SIZE: ", this.FileLength))
	printWithColor(Red, fmt.Sprint("Pieces size: ", this.PieceSize))
	printWithColor(Red, fmt.Sprint("Total pieces: ", this.TotalPieces))
	printWithColor(Red, fmt.Sprint("Block size: ", this.BlockLength))
	printWithColor(Red, fmt.Sprint("Amount of blocks: ", this.AmountOfBlocks))
	if this.FileLength == 0 {
		log.Panic("ERROR ON READING THE FILE LENGTH, FOR NOW THIS ONLY SUPPORTS SINGLE FILE DOWNLOADING")
	}

}

func (this *TorrentFileToBuild) loadTrackers(torrentInfo *TorrentFileInfo) {
	this.MainTracker = torrentInfo.Announce
	for _, tracker := range torrentInfo.AnnounceList {
		trackerToStr := strings.Join(tracker, "")
		this.ListOfTrackers = append(this.ListOfTrackers, trackerToStr)
	}
}

// Loop all the torrent trackers and get the peers that have the file
func (this *TorrentFileToBuild) getPeers() {

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
			//GET ALL THE PEERS THAT HAVE THE FILE FROM THE TRACKERS
			trackerAnnounceResponse, _, err := getPeers(
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

			//Add all the Ips and ports to the TorrentFileToBuild
			this.AddIpsThatHaveTheFile(ipsAndPorts)
			//CLOSE THE CONNECTION
			conn.Close()
		}
	}
	//time.Sleep(10 * time.Second)

}

func (this *TorrentFileToBuild) downloadFile() {
	//loop all the pieces
	for fileIndex, _ := range this.ListOfHashes {
		wholePiece := []byte{}

		for i := 0; i < this.BlockLength; i++ {
			blockOffset := i * this.BlockLength

			data, err := this.askPeerForBlockOfFile(fileIndex, this.InfoHash, blockOffset) //HARD CODED
			if err != nil {
				printWithColor(Red, err.Error())
			} else {
				wholePiece = append(wholePiece, data...)
			}
		}
		this.File[fileIndex] = wholePiece
	}
}

func (this *TorrentFileToBuild) askPeerForBlockOfFile(fileIndex int, infoHash []byte, blockOffset int) ([]byte, error) {

	for _, ip := range this.ipsWithTheFile {
		peerID, err := generatePeerID()
		if err != nil {
			log.Print("Error generating peerID", err)
		}

		file, err := connectToPeerAndRequestFile(ip, fileIndex, infoHash, peerID, blockOffset, this.BlockLength)
		if err != nil {
			log.Println(err)
			continue
		}
		printWithColor(Yellow, fmt.Sprint("ME DIERON LA FILE:!!!    ", file))
	}

	return nil, nil
}

// Receives the Ips Parsed as strings ("192.34.50.91:2092") and adds them to the TorrentFileToBuild.ipsWithTheFile Slice
func (this *TorrentFileToBuild) AddIpsThatHaveTheFile(ipsParsed []string) {
	for _, v := range ipsParsed {
		printWithColor(Red, v)
		this.ipsWithTheFile = append(this.ipsWithTheFile, v)
	}

}
