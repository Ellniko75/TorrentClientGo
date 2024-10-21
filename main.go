package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"log"
	"os"
	"os/exec"
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

// serves to unmarshal the torrent data
type TorrentFileInfo struct {
	Announce string `bencode:"announce"`
	Info     struct {
		Pieces      string `bencode:"pieces"`
		PieceLength int    `bencode:"piece length"`
	} `bencode:"info"`
	AnnounceList [][]string `bencode:"announce-list"` // Optional multiple trackers
}

func main() {
	//go listenConnections()

	time.Sleep(2 * time.Second)

	file, err := os.Open("./torrents/xoka.torrent")
	defer file.Close()
	if err != nil {
		log.Println("error on reading file")
	}

	//unmarshal the torrent info
	var torrentInfo TorrentFileInfo
	err = bencode.Unmarshal(file, &torrentInfo)
	if err != nil {
		log.Println("Error on unmarshaling")
	}

	//torrent that will be constructed
	TorrentFileToBuild := TorrentFileToBuild{}

	//don't try to do this on a function, it destroys itself lmao
	hexHash, err := getHexHash("db.torrent")
	if err != nil {
		log.Println(err)
	}
	hash, err := hex.DecodeString(hexHash)
	if err != nil {
		log.Println(err)
	}

	TorrentFileToBuild.loadInfoHash(hash)
	TorrentFileToBuild.loadHashes(&torrentInfo)
	TorrentFileToBuild.loadTrackers(&torrentInfo)
	TorrentFileToBuild.getPeers()
	//TorrentFileToBuild.downloadFile()

}
func getHexHash(torrentName string) (string, error) {
	cmd := exec.Command("python", "PythonScripts/CalculateHash.py", torrentName)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	hexHash := string(output)
	return hexHash, nil
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
