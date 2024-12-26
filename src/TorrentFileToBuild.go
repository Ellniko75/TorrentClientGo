package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/jackpal/bencode-go"
)

// contains all the info and functions necessary to download the file
type TorrentFileToBuild struct {
	Name            string //name of the file
	PieceSize       int    //size of each piece to download
	TotalPieces     int
	BlockLength     int
	AmountOfBlocks  int
	MainTracker     string
	FileLength      int
	Finished        bool
	TotalDownloaded struct {
		Downloaded int64
		mu         sync.Mutex
	}
	Uploaded struct {
		Uploaded int64
		mu       sync.Mutex
	}
	Connections    Connections //slice of all peers that have the file
	ListOfTrackers []string    //list of all the trackers
	ListOfHashes   []Hash      //hashes for each piece of the file
	InfoHash       []byte
}

type Hash struct {
	Hash      []byte
	Completed bool
}
type Connections struct {
	Conns []Connection
	Mu    sync.Mutex
}
type Connection struct {
	Conn            net.Conn
	Ip              string
	Using           bool //currently using
	Healthy         bool //if the connection has not responded we mark its healthy as false
	Speed           int  //Speed of that connection
	UsedAtLeastOnce bool
}
type BencodedResponseHTTPTracker struct {
	Interval int    `bencode:"interval"`
	Peers    string `bencode:"peers"`
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
	this.Uploaded.Uploaded = 0
	this.TotalDownloaded.Downloaded = 0
	this.Finished = false
	printWithColor(Red, fmt.Sprint("FILE TOTAL SIZE: ", this.FileLength))
	printWithColor(Red, fmt.Sprint("Pieces size: ", this.PieceSize))
	printWithColor(Red, fmt.Sprint("Total pieces: ", this.TotalPieces+1)) //need to add +1 since its an index that starts counting form 0
	printWithColor(Red, fmt.Sprint("Block size: ", this.BlockLength))
	printWithColor(Red, fmt.Sprint("Amount of blocks: ", this.AmountOfBlocks))

}

// this is the same as CalculateTotalPiecesAndBlockLength but it receives the information separated instead of as a TorrentFileInfo pointer
func (this *TorrentFileToBuild) LoadMetaData(fileLength int, pieceSize int) {
	this.FileLength = fileLength
	this.PieceSize = pieceSize
	this.TotalPieces = this.FileLength / this.PieceSize
	this.BlockLength = 16384
	this.AmountOfBlocks = this.PieceSize / this.BlockLength //Calculate the amount of blocks per piece

	this.Uploaded.Uploaded = 0
	this.TotalDownloaded.Downloaded = 0
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
func (this *TorrentFileToBuild) GetPeers(firstTime bool) {
	var w sync.WaitGroup
	for _, tracker := range this.ListOfTrackers {
		w.Add(1)
		go func() {
			defer w.Done()
			if tracker[:3] == "udp" {
				//Adjust the format of the UDP tracker URL
				trackerURL := strings.TrimPrefix(tracker, "udp://") //you need to strip the udp:// from the tracker to resolve the address later
				trackerURL = strings.TrimSuffix(trackerURL, "/announce")
				//create udp connection for the UDP tracker
				conn, err := createUdpConnection(trackerURL)
				defer conn.Close()
				if err != nil {
					printWithColor(Red, err.Error())
					return
				}
				//Create random transaction ID
				transactionID := int32(rand.Int31())

				//Request to UDP TRACKER and read the response
				transactionIDResponse, connectionIDResponse, err := initiateUdpConnection(conn, transactionID)
				if err != nil {
					printWithColor(Red, err.Error())
					return
				}
				//GENERATE A RANDOM ID FOR THE REQUEST
				peerID, _ := generatePeerID()

				eventType := getEventTypeForUDPTracker(firstTime, this)
				left := (int64(this.FileLength) - this.TotalDownloaded.Downloaded)
				//GET ALL THE PEERS THAT HAVE THE FILE FROM THE TRACKERS
				trackerAnnounceResponse, _, err := getPeersFromUdp(
					conn,
					this.InfoHash,
					connectionIDResponse,
					transactionIDResponse,
					peerID,
					this.TotalDownloaded.Downloaded,
					left,
					this.Uploaded.Uploaded,
					eventType)

				if err != nil {
					printWithColor(Red, err.Error())
					return
				}

				//parse the tracker response
				TrackerResponseParsed := TrackerResponse{}
				TrackerResponseParsed.Create(trackerAnnounceResponse)
				TrackerResponseParsed.Print()
				ipsAndPorts := TrackerResponseParsed.getIpAndPorts()
				//we only add the ips and ports if they actually are responsive
				//create all the connections and add them to the slice
				for _, v := range ipsAndPorts {
					w.Add(1)
					go func() {
						defer w.Done()
						this.AddConnection(v, peerID)
					}()
				}
			} else {
				infoHash := url.QueryEscape(string(this.InfoHash))
				peerId, _ := generatePeerID()
				peerIdArrByte := []byte(peerId[:])
				port := 6881
				//load the url query params
				url := tracker
				url += fmt.Sprint("?info_hash=", infoHash)
				url += fmt.Sprint("&peer_id=", string(peerIdArrByte))
				url += fmt.Sprint("&ip=", "255.255.255.255")
				url += fmt.Sprint("&port=", port)
				url += fmt.Sprint("&downloaded=", this.TotalDownloaded.Downloaded)
				left := this.FileLength - int(this.TotalDownloaded.Downloaded)
				url += fmt.Sprint("&left=", left)
				url += fmt.Sprint("&uploaded=", this.Uploaded.Uploaded)
				if firstTime {
					event := "started"
					if this.TotalDownloaded.Downloaded == int64(this.FileLength) {
						event = "completed"
					}
					url += fmt.Sprint("&event=", event)
				}

				httpClient := http.Client{
					Timeout: 2 * time.Second,
				}
				resp, err := httpClient.Get(url)
				printWithColor(Yellow, "HTTP TRACKER RESPONDED")
				if err != nil {
					log.Println(err)
					return
				}
				printWithColor(Yellow, "HTTP tracker response got")
				//parse the request of the http tracker
				BencodedResponse := BencodedResponseHTTPTracker{}
				err = bencode.Unmarshal(resp.Body, &BencodedResponse)
				if err != nil {
					log.Println("Error unmarshaling the http response into bencoded data")
					return
				}
				//add the connections
				for i := 0; i < len(BencodedResponse.Peers); i += 6 {
					w.Add(1)
					go func() {
						defer w.Done()
						peersStr := BencodedResponse.Peers
						ip := fmt.Sprint(peersStr[i], ".", peersStr[i+1], ".", peersStr[i+2], ".", peersStr[i+3])
						port := string(peersStr[i+4]) + string(peersStr[i+5])
						portNumber := binary.BigEndian.Uint16([]byte(port))
						fullIp := fmt.Sprint(ip, ":", portNumber)
						this.AddConnection(fullIp, peerId)
					}()
				}
			}
		}()
	}
	w.Wait()
	printWithColor(Red, "FINISHED GET PEERS")
}
func (this *TorrentFileToBuild) pollGetPeersEveryCoupleMinutes() {
	go func() {
		for {
			time.Sleep(10 * time.Second)

			if this.allFilesAreDownloaded() {
				this.GetPeers(false)
				return
			} else {
				this.GetPeers(false)
			}
		}
	}()
}

// Creates the connections if they are not repeated and adds them to the slice
func (this *TorrentFileToBuild) AddConnection(ipAndPort string, peerID [20]byte) {
	fmt.Println(ipAndPort)
	if this.isIpRepeatedAndHealthy(ipAndPort) {
		fmt.Println("ip repeated")
		return
	}
	conn, err := initiatePeerConnection(ipAndPort, this.InfoHash, peerID)
	if err != nil {
		printWithColor(Red, fmt.Sprint("Could not establish connection with ", ipAndPort))
		return
	}
	printWithColor(Green, "PEER CONNECTION ADDED SUCCESSFULLY")
	this.Connections.Mu.Lock()
	this.Connections.Conns = append(this.Connections.Conns, Connection{Conn: conn, Using: false, Ip: ipAndPort, Healthy: true, UsedAtLeastOnce: false})
	this.Connections.Mu.Unlock()
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
	fmt.Println("STARTED DOWNLOAD")
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
			printWithColor(Gray, fmt.Sprint("IP WE ARE USING:", connectionToUse.Ip))
			//check if this is the final piece
			final := fileIndex == this.TotalPieces
			w.Add(1)
			go func(conn *Connection) {
				defer w.Done()
				//get the file piece, the one thats composed by all the blocks and check if the hash is correct
				data, err := this.askForFilePiece(fileIndex, v.Hash, conn, final)
				conn.Using = false
				if err != nil {
					printWithColor(Red, err.Error())
					WriteToErrorstxt(fmt.Sprint("Error with IP:", conn.Ip, err.Error()))
					return
				}
				printWithColor(Green, fmt.Sprint("Hash matched - FILE:", fileIndex, " | IP downloaded from:", conn.Ip))
				//set completed to true - need to do it like this, because v is a copy of the value and not a reference
				this.ListOfHashes[fileIndex].Completed = true
				//Write to the temp file the downloaded piece
				indexStart := this.PieceSize * fileIndex
				this.writeToTempFile(data, indexStart)
				//increment the total data downloaded
				this.TotalDownloaded.mu.Lock()
				this.TotalDownloaded.Downloaded += int64(len(data))
				this.TotalDownloaded.mu.Unlock()
			}(connectionToUse)
		}
		if this.allFilesAreDownloaded() {
			printWithColor(Red, "FINISHED")
			this.Finished = true
			break
		}
		w.Wait()
	}
}

// runs on main thread, constantly checking if there are any connection up for use
func (this *TorrentFileToBuild) GetUnusedConnection() *Connection {
	for {
		conns := this.Connections.Conns
		for i := 0; i < len(conns); i++ {
			if conns[i].Using == false && conns[i].Healthy {
				return &conns[i]
			}
			time.Sleep(1 * time.Millisecond)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// Requests the file piece and checks if the hash is okay
func (this *TorrentFileToBuild) askForFilePiece(fileIndex int, fileHash []byte, connectionToUse *Connection, Final bool) ([]byte, error) {
	//download the piece
	data, err := connectToPeerAndRequestWholePiece(connectionToUse, fileIndex, this, Final)
	//if there was an error mark the connection as unhealthy
	if err != nil {
		connectionToUse.Healthy = false
		connectionToUse.Conn.Close()
		return nil, err
	}

	//hash of the whole piece gotten
	wholePieceSha1Hash := GetSha1Hash(data)
	if reflect.DeepEqual(fileHash, wholePieceSha1Hash) {
		//WriteToOkstxt(fileIndex)
		return data, nil
	}
	if len(data) > 5 {
		//hash of the whole piece except the first 5 bytes (sometimes this can help)
		alternativeHash := GetSha1Hash(data[5:])
		if reflect.DeepEqual(fileHash, alternativeHash) {
			//WriteToOkstxt(fileIndex)
			return data[5:], nil
		}
	}
	return nil, createError("askForFilePiece()", fmt.Sprint("THE HASH DIDN'T MATCH, FILE: ", fileIndex, " LENGTH GOTTEN: ", len(data)))
}

// gets the partial.bin file that holds all the binary data, and writes it in chunks to the desired path and with the desired name
func (this *TorrentFileToBuild) writeFileInPieces(directory string, name string, pStart int, pEnd int) {
	//open the file we read from
	fileRead, err := os.Open("partial.bin")
	if err != nil {
		log.Println(err)
	}
	defer fileRead.Close()
	//create the file we are going to write to
	fileWriteTo, err := os.Create(fmt.Sprint(directory, name))
	if err != nil {
		log.Println(err)
	}
	defer fileWriteTo.Close()

	startIndexWrite := 0
	startIndexRead := pStart
	chunkWriteReadLength := 10000000
	leftToWrite := pEnd - pStart
	for {
		//if we don't have any more to write-read, we break
		if leftToWrite <= 0 {
			break
		}
		//if the chunk we are going to read surpasses the "left we have to read", we set it to what we have left
		if chunkWriteReadLength > leftToWrite {
			chunkWriteReadLength = leftToWrite
		}
		//holds in ram the data
		toWrite := make([]byte, chunkWriteReadLength)
		//read the data from the start
		fileRead.Seek(int64(startIndexRead), 0)
		_, err = fileRead.Read(toWrite)
		if err != nil {
			log.Println(err)
		}
		//write it from the start
		_, err = fileWriteTo.Seek(int64(startIndexWrite), 0)
		if err != nil {
			log.Println("error at seeking")
		}
		_, err = fileWriteTo.Write(toWrite)
		if err != nil {
			log.Println(err)
		}
		//set the new index for reading
		startIndexRead += chunkWriteReadLength
		//set the new index for writing
		startIndexWrite += chunkWriteReadLength
		//now we have less leftToWrite
		leftToWrite -= chunkWriteReadLength
	}
}

// calls writeFileInPieces, and just does some extra things before
func (this *TorrentFileToBuild) writePieceOfFileToDisk(fullDirectory string, start int, end int) error {
	//ensure the path exists, if not create it
	toArr := strings.Split(fullDirectory, "/")
	//get the path where the file will be saved
	path := strings.Join(toArr[:len(toArr)-1], "/")
	//create the directory if it does not exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, 0700)
	}
	name := strings.Join(toArr[len(toArr)-1:], "")
	this.writeFileInPieces(fmt.Sprint(path, "/"), name, start, end)
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

func (this *TorrentFileToBuild) tempFileInit() []byte {
	partialfileRead := make([]byte, this.FileLength)

	return partialfileRead
}

func (this *TorrentFileToBuild) writeTempFile(data []byte) error {
	err := os.WriteFile("./partial.bin", data, 0644)
	if err != nil {
		return err
	}
	return nil
}

// get a byte array and write it to the specified indices
func (this *TorrentFileToBuild) writeToTempFile(dataToWrite []byte, start int) {

	file, err := os.OpenFile("partial.bin", os.O_RDWR, 0644)
	defer file.Close()

	if err != nil {
		log.Println("ERROR ON WRITEPARTIALFILESTARTEND")
	}
	//move to the place where you want to write the data
	_, err = file.Seek(int64(start), 0)
	if err != nil {
		log.Println("error seeking the partial file ", err)
	}
	//write the data
	_, err = file.Write(dataToWrite)
	if err != nil {
		log.Println("error on writing partial file ", err)
	}
	printWithColor(Blue, "Written to partial successfully")

}
func (this *TorrentFileToBuild) getTempFile() ([]byte, error) {
	data, err := os.ReadFile("partial.bin")
	if err != nil {
		return nil, err
	}
	return data, err
}
func (this *TorrentFileToBuild) getTempFileStartAndEnd(start int64, end int64) ([]byte, error) {
	file, err := os.Open("partial.bin")
	defer file.Close()
	if err != nil {
		return nil, err
	}
	readArrLength := end - start
	dataArr := make([]byte, readArrLength)
	file.Seek(start, 0)
	file.Read(dataArr)

	return dataArr, nil
}
func (this *TorrentFileToBuild) deleteTempFile() {
	err := os.Remove("partial.bin")
	if err != nil {
		log.Println(err)
	}
}

func (this *TorrentFileToBuild) shareCurrentTorrent() {

	file, err := os.Open("partial.bin")
	if err != nil {
		log.Println("NO SE PUDO LEER PARTIAL.BIN")
	}
	defer file.Close()

	l, err := net.Listen("tcp", ":6881")
	printWithColor(Green, "LISTENING FOR INCOMING CONNECTIONS")
	for {
		conn, err := l.Accept()
		if err != nil {
			printWithColor(Red, fmt.Sprint("ERROR EN ACEPTAR CONEXION INCOMING:", err))
		}
		go this.HandleConnection(conn)
	}
}
func (this *TorrentFileToBuild) HandleConnection(currentCon net.Conn) {
	for {
		buf := make([]byte, 1000)
		n, err := currentCon.Read(buf)
		if err != nil {
			printWithColor(Red, fmt.Sprint("Error on handling incoming connection:", err))
		}
		if n == 0 {
			printWithColor(Green, "0 BYTES RECEIVED")
		}
		printWithColor(Green, fmt.Sprint("RECEIVED:", buf[:n]))
		protocolMsgLength := binary.BigEndian.Uint16(buf[:1])
		protocolMessage := buf[1:protocolMsgLength]
		fmt.Println("PROTOCOL MESSAGE:", protocolMessage)
	}
}
