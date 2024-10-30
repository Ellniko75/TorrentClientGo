package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"
)

// contains all the info and functions necessary to download the file
type TorrentFileToBuild struct {
	PieceSize      int //size of each piece to download
	TotalPieces    int
	BlockLength    int
	AmountOfBlocks int
	Connections    Connections //slice of all peers that have the file
	MainTracker    string
	ListOfTrackers []string //list of all the trackers
	ListOfHashes   []Hash   //hashes for each piece of the file
	InfoHash       []byte
	FileLength     int
	File           [100000][]byte //property to write the file when the pieces arrive

}

type Hash struct {
	Hash      []byte
	Completed bool
}

type Connections struct {
	Conns []Connection
	mu    sync.Mutex
}
type Connection struct {
	Conn    net.Conn
	Ip      string
	Using   bool //currently using
	Healthy bool //if the connection has not responded we mark its healthy as false
	mu      sync.Mutex
}

// TODO: for now it loads the infohash forcefully, because this library is complete shit and cannot for the life of it calculate the infohash
func (this *TorrentFileToBuild) LoadPieceHashes(torrentInfo *TorrentFileInfo) {
	hashLen := 20 //sha1 length
	for i := 0; i < len(torrentInfo.Info.Pieces); i += hashLen {
		currentHash := torrentInfo.Info.Pieces[i : i+hashLen]
		this.ListOfHashes = append(this.ListOfHashes, Hash{Hash: []byte(currentHash), Completed: false})
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
func (this *TorrentFileToBuild) GetPeers() {

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

			//Create random transaction ID
			transactionID := int32(rand.Int31())

			//Request to UDP TRACKER and read the response
			transactionIDResponse, connectionIDResponse, err := initiateUdpConnection(conn, transactionID)
			if err != nil {
				printWithColor(Red, err.Error())
				continue
			}

			//GENERATE A RANDOM ID FOR THE REQUEST
			peerID, _ := generatePeerID()

			//GET ALL THE PEERS THAT HAVE THE FILE FROM THE TRACKERS
			trackerAnnounceResponse, _, err := getPeersFromUdp(
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

			printWithColor(Green, "ADDING NEW PEERS...")
			//we only add the ips and ports if they actually are responsive

			//create all the connections and add them to the slice
			var w sync.WaitGroup
			for _, v := range ipsAndPorts {
				go func() {
					w.Add(1)
					defer w.Done()
					this.AddConnection(v, peerID)

				}()
			}
			w.Wait()
			//CLOSE THE CONNECTION
			conn.Close()
		}
	}
}

// Creates the connections if they are not repeated and adds them to the slice
func (this *TorrentFileToBuild) AddConnection(ipAndPort string, peerID [20]byte) {

	if this.isIpRepeated(ipAndPort) {
		return
	}
	conn, err := initiatePeerConnection(ipAndPort, this.InfoHash, peerID)
	if err != nil {
		return
	}
	this.Connections.Conns = append(this.Connections.Conns, Connection{Conn: conn, Using: false, Ip: ipAndPort, Healthy: true})
}
func (this *TorrentFileToBuild) isIpRepeated(ip string) bool {
	for i := 0; i < len(this.Connections.Conns); i++ {
		currentIP := this.Connections.Conns[i].Ip
		if ip == currentIP {
			return true
		}
	}

	return false
}

// Blocks form a Piece, and Pieces form the file
func (this *TorrentFileToBuild) downloadFileAsync() {
	var w sync.WaitGroup
	//loop all the pieces and request them
	for fileIndex, v := range this.ListOfHashes {
		if v.Completed {
			continue
		}
		//get any connection that is not being currently used
		connectionToUse := this.GetUnusedConnection()
		connectionToUse.mu.Lock()
		connectionToUse.Using = true

		go func() {
			w.Add(1)
			defer w.Done()
			printWithColor(Blue, fmt.Sprint("Downloading Piece: ", fileIndex))

			final := fileIndex == this.TotalPieces

			//get the file piece, the one thats composed by all the blocks and check if the hash is correct
			data, err := this.askForFilePiece(fileIndex, v.Hash, connectionToUse, final)
			if err != nil {
				printWithColor(Red, err.Error())
				WriteToErrorstxt(fileIndex)
				return
			}
			printWithColor(Green, fmt.Sprint("Downloaded piece: ", fileIndex))
			printWithColor(Green, fmt.Sprint(" Hash match on file ", " fileIndex"))
			time.Sleep(3 * time.Second)
			//set completed to true
			v.Completed = true
			this.File[fileIndex] = data
		}()

		//start := fileIndex * 131072
		//expectedFile := GetExpectedFile()[start : start+131072]
		//gotten := data
		//startOfDiscrepancy := CheckPlacesWhereTheBytesAreDifferent(expectedFile, gotten[5:])
		//record the errors
		//fmt.Println("length expected: ", len(expectedFile))
		//fmt.Println("length gotten: ", len(data))
		//time.Sleep(2 * time.Second)
	}
	w.Wait()

}

/*
func (this *TorrentFileToBuild) downloadFile() {

	//loop all the pieces and request them
	for fileIndex, v := range this.ListOfHashes {
		if v.Completed {
			continue
		}
		//get any connection that is not being currently used
		connectionToUse := this.GetUnusedConnection()
		connectionToUse.Using = true
		connectionToUse.mu.Lock()

		printWithColor(Blue, fmt.Sprint("Downloading Piece: ", fileIndex))

		//get the file piece, the one thats composed by all the blocks and check if the hash is correct
		data, err := this.askForFilePiece(fileIndex, v.Hash, connectionToUse, false)
		if err != nil {
			printWithColor(Red, err.Error())
			WriteToErrorstxt(fileIndex)
			continue
		}
		printWithColor(Green, fmt.Sprint("Downloaded piece: ", fileIndex))
		printWithColor(Green, fmt.Sprint(" Hash match on file ", " fileIndex"))
		//set completed to true
		v.Completed = true
		this.File[fileIndex] = data

		//start := fileIndex * 131072
		//expectedFile := GetExpectedFile()[start : start+131072]
		//gotten := data
		//startOfDiscrepancy := CheckPlacesWhereTheBytesAreDifferent(expectedFile, gotten[5:])
		//record the errors
		//fmt.Println("length expected: ", len(expectedFile))
		//fmt.Println("length gotten: ", len(data))
		//time.Sleep(2 * time.Second)
	}

}*/

func (this *TorrentFileToBuild) DownloadMissingPieces() {

}

// runs on main thread, constantly checking if there are any connection up for use
func (this *TorrentFileToBuild) GetUnusedConnection() *Connection {

	for {
		conns := this.Connections.Conns
		for i := 0; i < len(conns); i++ {
			if !conns[i].Using && conns[i].Healthy {

				return &conns[i]
			}
		}
		time.Sleep(1 * time.Millisecond)
	}
}

// Requests the file piece and checks if the hash is okay
func (this *TorrentFileToBuild) askForFilePiece(fileIndex int, fileHash []byte, connectionToUse *Connection, Final bool) ([]byte, error) {
	//download the piece
	data, err := connectToPeerAndRequestWholePiece(connectionToUse, fileIndex, this, Final)
	connectionToUse.mu.Unlock()
	connectionToUse.Using = false
	if err != nil {
		connectionToUse.Healthy = false
		log.Println("askForFilePiece()", err)
		return nil, err
	}

	//hash of the whole piece gotten
	wholePieceSha1Hash := GetSha1Hash(data)
	if reflect.DeepEqual(fileHash, wholePieceSha1Hash) {
		WriteToOkstxt(fileIndex)
		return data, nil
	}
	if len(data) > 5 {
		//hash of the whole piece except the first 5 bytes (sometimes this can help)
		alternativeHash := GetSha1Hash(data[5:])
		if reflect.DeepEqual(fileHash, alternativeHash) {
			WriteToOkstxt(fileIndex)
			return data[5:], nil
		}
	}

	return nil, createError("askForFilePiece()", fmt.Sprint("THE HASH DIDN'T MATCH, FILE: ", fileIndex, " LENGTH GOTTEN: ", len(data)))
}

func generatePeerID() ([20]byte, error) {

	var peerId bytes.Buffer

	firstPart := []byte("-Go1234-")
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
