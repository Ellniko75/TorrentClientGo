package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"
)

// contains all the info and functions necessary to download the file
type TorrentFileToBuild struct {
	Name           string //name of the file
	PieceSize      int    //size of each piece to download
	TotalPieces    int
	BlockLength    int
	AmountOfBlocks int
	Connections    Connections //slice of all peers that have the file
	MainTracker    string
	ListOfTrackers []string //list of all the trackers
	ListOfHashes   []Hash   //hashes for each piece of the file
	InfoHash       []byte
	FileLength     int
	File           [10000000][]byte //property to write the file when the pieces arrive
	WholeFile      []byte
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

func (this *TorrentFileToBuild) LoadPieceHashes(torrentInfo *TorrentFileInfo) {
	hashLen := 20 //sha1 length
	for i := 0; i < len(torrentInfo.Info.Pieces); i += hashLen {
		currentHash := torrentInfo.Info.Pieces[i : i+hashLen]
		this.ListOfHashes = append(this.ListOfHashes, Hash{Hash: []byte(currentHash), Completed: false})
	}
}

func (this *TorrentFileToBuild) LoadInfoHash(hash []byte) {
	this.InfoHash = hash
}
func (this *TorrentFileToBuild) LoadName(name string) {
	this.Name = name
}
func (this *TorrentFileToBuild) CalculateTotalPiecesAndBlockLength(info *TorrentFileInfo) {
	this.FileLength = info.Info.Length
	this.PieceSize = info.Info.PieceLength
	this.TotalPieces = this.FileLength / this.PieceSize
	this.BlockLength = 16384
	this.AmountOfBlocks = this.PieceSize / this.BlockLength //Calculate the amount of blocks per piece

	printWithColor(Red, fmt.Sprint("FILE TOTAL SIZE: ", this.FileLength))
	printWithColor(Red, fmt.Sprint("Pieces size: ", this.PieceSize))
	printWithColor(Red, fmt.Sprint("Total pieces: ", this.TotalPieces+1)) //need to add +1 since its an index that starts counting form 0
	printWithColor(Red, fmt.Sprint("Block size: ", this.BlockLength))
	printWithColor(Red, fmt.Sprint("Amount of blocks: ", this.AmountOfBlocks))
	//if this.FileLength == 0 {
	//	log.Panic("ERROR ON READING THE FILE LENGTH, FOR NOW THIS ONLY SUPPORTS SINGLE FILE DOWNLOADING")
	//}
}

// this is the same as CalculateTotalPiecesAndBlockLength but it receives the information separated instead of as a TorrentFileInfo pointer
func (this *TorrentFileToBuild) LoadMetaData(fileLength int, pieceSize int) {
	this.FileLength = fileLength
	this.PieceSize = pieceSize
	this.TotalPieces = this.FileLength / this.PieceSize
	this.BlockLength = 16384
	this.AmountOfBlocks = this.PieceSize / this.BlockLength //Calculate the amount of blocks per piece

	printWithColor(Red, fmt.Sprint("FILE TOTAL SIZE: ", this.FileLength))
	printWithColor(Red, fmt.Sprint("Pieces size: ", this.PieceSize))
	printWithColor(Red, fmt.Sprint("Total pieces: ", this.TotalPieces+1)) //need to add +1 since its an index that starts counting form 0
	printWithColor(Red, fmt.Sprint("Block size: ", this.BlockLength))
	printWithColor(Red, fmt.Sprint("Amount of blocks: ", this.AmountOfBlocks))
}

func (this *TorrentFileToBuild) LoadTrackers(torrentInfo *TorrentFileInfo) {
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
			defer conn.Close()
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
			this.Connections.mu.Lock()
			for _, v := range ipsAndPorts {
				go func() {
					w.Add(1)
					defer w.Done()
					this.AddConnection(v, peerID)
				}()
			}
			w.Wait()
			this.Connections.mu.Unlock()
		}
	}
}
func (this *TorrentFileToBuild) pollGetPeersEveryCoupleMinutes() {
	go func() {
		for {
			if this.allFilesAreDownloaded() {
				return
			}

			this.GetPeers()

			time.Sleep(10 * time.Second)

		}

	}()
}

// Creates the connections if they are not repeated and adds them to the slice
func (this *TorrentFileToBuild) AddConnection(ipAndPort string, peerID [20]byte) {
	if this.isIpRepeatedAndHealthy(ipAndPort) {
		return
	}
	conn, err := initiatePeerConnection(ipAndPort, this.InfoHash, peerID)
	if err != nil {
		return
	}
	this.Connections.Conns = append(this.Connections.Conns, Connection{Conn: conn, Using: false, Ip: ipAndPort, Healthy: true})
}
func (this *TorrentFileToBuild) isIpRepeatedAndHealthy(ip string) bool {
	for i := 0; i < len(this.Connections.Conns); i++ {
		currentIP := this.Connections.Conns[i].Ip
		isHealthy := this.Connections.Conns[i].Healthy
		if ip == currentIP && isHealthy {
			return true
		}
	}
	return false
}
func (this *TorrentFileToBuild) allFilesAreDownloaded() bool {
	for _, v := range this.ListOfHashes {
		if !v.Completed {
			return false
		}
	}
	return true
}

// Blocks form a Piece, and Pieces form the file
func (this *TorrentFileToBuild) downloadFileAsync() {
	var w sync.WaitGroup
	for {
		//loop all the pieces and request them
		for fileIndex, v := range this.ListOfHashes {
			if v.Completed {
				continue
			}
			//get any connection that is not being currently used
			connectionToUse := this.GetUnusedConnection()
			connectionToUse.Using = true
			go func() {
				w.Add(1)
				defer w.Done()
				//check if this is the final piece
				final := fileIndex == this.TotalPieces
				//get the file piece, the one thats composed by all the blocks and check if the hash is correct
				data, err := this.askForFilePiece(fileIndex, v.Hash, connectionToUse, final)
				if err != nil {
					printWithColor(Red, err.Error())
					WriteToErrorstxt(fileIndex)
					return
				}
				//Show completed message
				printWithColor(Green, fmt.Sprint(" Hash match on file ", fileIndex))
				//set completed to true - need to do it like this, because v is a copy of the value and not a reference
				this.ListOfHashes[fileIndex].Completed = true
				this.File[fileIndex] = data
			}()
		}
		if this.allFilesAreDownloaded() {
			break
		}
	}

	w.Wait()
	//get the pieces of all the file and store it in WholePiece
	data := this.File[:this.TotalPieces+1]
	for _, v := range data {
		this.WholeFile = append(this.WholeFile, v...)
	}
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
	connectionToUse.Using = false
	//if there was an error mark the connection as unhealthy
	if err != nil {
		connectionToUse.Healthy = false
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
func (this *TorrentFileToBuild) writeFileToDisk(directory string) error {

	err := os.WriteFile(fmt.Sprint(directory, this.Name), this.WholeFile, 0644)
	//fmt.Println("NAME: ", this.Name)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// this one needs the whole path to write the file, including the file name
func (this *TorrentFileToBuild) writePieceOfFileToDisk(fullDirectory string, from int, end int) error {
	//ensure the path exists, if not create it
	toArr := strings.Split(fullDirectory, "/")
	//get the path where the file will be saved
	pathWithoutTheFileName := strings.Join(toArr[:len(toArr)-1], "/")
	if _, err := os.Stat(pathWithoutTheFileName); os.IsNotExist(err) {
		err = os.MkdirAll(pathWithoutTheFileName, 0700)
		//fmt.Println("Created the directory: ", pathWithoutTheFileName)
	}
	err := os.WriteFile(fmt.Sprint(fullDirectory), this.WholeFile[from:end], 0644)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
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
