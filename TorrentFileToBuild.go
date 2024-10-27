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
	ipsWithTheFile []string //slice of all peers that have the file
	MainTracker    string
	ListOfTrackers []string //list of all the trackers
	ListOfHashes   [][]byte //hashes for each piece of the file
	InfoHash       []byte
	FileLength     int
	File           [100000][]byte //property to write the file when the pieces arrive
	Connections    []net.Conn     //this connections are initialized by the CreateConnections() function, the peers all store the infohash, so once you send it on the handhsake its not necessary to do so again
}

// TODO: for now it loads the infohash forcefully, because this library is complete shit and cannot for the life of it calculate the infohash
func (this *TorrentFileToBuild) loadHashes(torrentInfo *TorrentFileInfo) {

	hashLen := 20 //sha1 length
	for i := 0; i < len(torrentInfo.Info.Pieces); i += hashLen {
		currentHash := torrentInfo.Info.Pieces[i : i+hashLen]
		this.ListOfHashes = append(this.ListOfHashes, []byte(currentHash))
	}
}

func (this *TorrentFileToBuild) CreateConnections() {
	printWithColor(Blue, " Creating Connections")
	var wg sync.WaitGroup

	for _, v := range this.ipsWithTheFile {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := createTcpConnection(v)
			if err != nil {
				printWithColor(Red, fmt.Sprint("Error on CreateConnections ", err.Error()))
				return
			}
			PeerId, err := generatePeerID()
			if err != nil {
				printWithColor(Red, "Error generating peer id")
				return
			}
			//send the handshake
			_, err = handleHandshake(this.InfoHash, PeerId, conn)
			if err != nil {
				printWithColor(Red, "Error on handshake")
				return
			}
			fmt.Println("Connection established")
			this.Connections = append(this.Connections, conn)
		}()
		/*
			conn, err := createTcpConnection(v)
			if err != nil {
				printWithColor(Red, fmt.Sprint("Error on CreateConnections ", err.Error()))
				continue
			}
			PeerId, err := generatePeerID()
			if err != nil {
				printWithColor(Red, "Error generating peer id")
				continue
			}
			//send the handshake
			_, err = handleHandshake(this.InfoHash, PeerId, conn)
			if err != nil {
				printWithColor(Red, "Error on handshake")
				continue
			}
			fmt.Println("Connection established")
			this.Connections = append(this.Connections, conn)*/
	}
	wg.Wait()

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

// Blocks form a Piece, and Pieces form the file
func (this *TorrentFileToBuild) downloadFile() {
	//loop all the pieces
	for fileIndex, fileHash := range this.ListOfHashes {
		printWithColor(Blue, fmt.Sprint("Downloading Piece: ", fileIndex))
		//get the file piece, the one thats composed by all the blocks and check if the hash is correct
		data, err := this.askForFilePiece(fileIndex, fileHash)
		if err != nil {
			printWithColor(Red, err.Error())
			WriteToErrorstxt(fileIndex)
			continue
		}
		printWithColor(Green, fmt.Sprint("Downloaded piece: ", fileIndex))
		printWithColor(Green, fmt.Sprint(" Hash match on file ", " fileIndex"))

		//start := fileIndex * 131072
		//expectedFile := GetExpectedFile()[start : start+131072]
		//gotten := data
		//startOfDiscrepancy := CheckPlacesWhereTheBytesAreDifferent(expectedFile, gotten[5:])
		//record the errors
		//fmt.Println("length expected: ", len(expectedFile))
		//fmt.Println("length gotten: ", len(data))
		//time.Sleep(2 * time.Second)
		this.File[fileIndex] = data
	}

}

// loops all the connections and request the piece
func (this *TorrentFileToBuild) askForFilePiece(fileIndex int, fileHash []byte) ([]byte, error) {
	for _, conn := range this.Connections {
		data, err := connectToPeerAndRequestWholePiece(conn, fileIndex, this.BlockLength, this.AmountOfBlocks)
		if err != nil {
			log.Println("askForFilePiece()", err)
			continue
		}

		//hash of the whole piece gotten
		wholePieceSha1Hash := GetSha1Hash(data)
		if reflect.DeepEqual(fileHash, wholePieceSha1Hash) {
			WriteToOkstxt(fileIndex)
			return data, nil
		}
		//hash of the whole piece except the first 5 bytes (sometimes this can help)
		alternativeHash := GetSha1Hash(data[5:])
		if reflect.DeepEqual(fileHash, alternativeHash) {
			WriteToOkstxt(fileIndex)
			return data[5:], nil
		}
	}
	return nil, createError("askForFilePiece()", " Unable to establish any connections or the hash didn't match")
}

// Receives the Ips Parsed as strings ("192.34.50.91:2092") and adds them to the TorrentFileToBuild.ipsWithTheFile Slice
func (this *TorrentFileToBuild) AddIpsThatHaveTheFile(ipsParsed []string) {
	for _, v := range ipsParsed {
		printWithColor(Red, v)
		this.ipsWithTheFile = append(this.ipsWithTheFile, v)
	}

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
