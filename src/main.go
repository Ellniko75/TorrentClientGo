package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

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
		Name        string `bencode:"name"`
		Files       []struct {
			Length int      `bencode:"length"`
			Path   []string `bencode:"path"`
		} `bencode:"files"`
	} `bencode:"info"`
	AnnounceList [][]string `bencode:"announce-list"`
}

func main() {
	ResetOksAndErrors()
	torrentUrl := "../torrents/torrentCustom.torrent"

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
	//don't try to do all TorrentFileToBuild on a single function, it destroys itself lmao
	hexHash, err := getHexHash(torrentUrl)
	if err != nil {
		log.Println(err)
	}
	hash, err := hex.DecodeString(hexHash)
	if err != nil {
		log.Println(err)
	}
	printWithColor(Yellow, fmt.Sprint("HASH: ", hash))

	torrentIsSingleFile := torrentInfo.Info.Length > 0

	if torrentIsSingleFile {
		//torrent that will be constructed
		TorrentFileToBuild := TorrentFileToBuild{}
		TorrentFileToBuild.LoadInfoHash(hash)
		TorrentFileToBuild.LoadName(torrentInfo.Info.Name)
		TorrentFileToBuild.LoadPieceHashes(&torrentInfo)
		TorrentFileToBuild.LoadTrackers(&torrentInfo)
		TorrentFileToBuild.CalculateTotalPiecesAndBlockLength(&torrentInfo)
		TorrentFileToBuild.GetPeers()
		TorrentFileToBuild.pollGetPeersEveryCoupleMinutes()
		TorrentFileToBuild.downloadFileAsync()
		TorrentFileToBuild.writeFileToDisk("../output/")
	} else {
		//handle multiple file download
		totalSizeOfFile := getTotalSizeOfMulitpleFilesTorrent(&torrentInfo)
		//torrent that will be constructed
		TorrentFileToBuild := TorrentFileToBuild{}
		TorrentFileToBuild.LoadInfoHash(hash)
		TorrentFileToBuild.LoadPieceHashes(&torrentInfo)
		TorrentFileToBuild.LoadTrackers(&torrentInfo)
		TorrentFileToBuild.LoadMetaData(totalSizeOfFile, torrentInfo.Info.PieceLength)
		TorrentFileToBuild.GetPeers()
		TorrentFileToBuild.pollGetPeersEveryCoupleMinutes()
		TorrentFileToBuild.downloadFileAsync()

		//write each file to the disk
		start := 0
		for _, v := range torrentInfo.Info.Files {
			end := start + v.Length
			currentFilePath := strings.Join(v.Path, "/")
			TorrentFileToBuild.writePieceOfFileToDisk(fmt.Sprint("../output/", currentFilePath), start, end)
			start = end
		}
	}
}

func getHexHash(torrentPath string) (string, error) {
	cmd := exec.Command("python", "../PythonScripts/CalculateHash.py", torrentPath)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	hexHash := string(output)
	return hexHash, nil
}

// returns the total length of all the files combined
func getTotalSizeOfMulitpleFilesTorrent(torrentInfo *TorrentFileInfo) int {
	totalFileSize := 0
	for _, v := range torrentInfo.Info.Files {
		totalFileSize += v.Length
	}
	return totalFileSize
}
