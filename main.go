package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/exec"

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
		Length      int    `bencode:"length"`
	} `bencode:"info"`
	AnnounceList [][]string `bencode:"announce-list"` // Optional multiple trackers
}

func main() {

	ResetOksAndErrors()
	torrentUrl := "./torrents/xoka.torrent"

	file, err := os.Open(torrentUrl)
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

	//don't try to do all this on a single function, it destroys itself lmao
	hexHash, err := getHexHash(torrentUrl)
	if err != nil {
		log.Println(err)
	}
	hash, err := hex.DecodeString(hexHash)
	if err != nil {
		log.Println(err)
	}

	TorrentFileToBuild.loadInfoHash(hash)
	TorrentFileToBuild.LoadPieceHashes(&torrentInfo)
	TorrentFileToBuild.loadTrackers(&torrentInfo)
	TorrentFileToBuild.CalculateTotalPiecesAndBlockLength(torrentInfo)
	TorrentFileToBuild.GetPeers()

	TorrentFileToBuild.downloadFileAsync()

	data := TorrentFileToBuild.File[:555]

	ActualFile := []byte{}
	for _, v := range data {
		ActualFile = append(ActualFile, v...)
	}

	fmt.Print("total length of downloaded: ", len(ActualFile))

	err = os.WriteFile("xokados.mp4", ActualFile, 0644)

	//TorrentFileToBuild.downloadFile()

}

func getHexHash(torrentPath string) (string, error) {
	cmd := exec.Command("python", "PythonScripts/CalculateHash.py", torrentPath)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	hexHash := string(output)
	return hexHash, nil
}
