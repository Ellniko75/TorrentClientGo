package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"torrent/pythonScripts"

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

	torrentPath := "../torrents/torrentCustom.torrent"

	file, err := os.Open(torrentPath)
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

	hash, err := pythonScripts.GetInfoHash(torrentPath)
	printWithColor(Blue, fmt.Sprint("HASH: ", hash))

	torrentIsSingleFile := torrentInfo.Info.Length > 0
	if torrentIsSingleFile {
		TorrentFileToBuild := TorrentFileToBuild{}
		TorrentFileToBuild.LoadInfoHash(hash)
		TorrentFileToBuild.LoadName(torrentInfo.Info.Name)
		TorrentFileToBuild.LoadPieceHashes(&torrentInfo)
		TorrentFileToBuild.LoadTrackers(&torrentInfo)
		TorrentFileToBuild.CalculateTotalPiecesAndBlockLength(&torrentInfo)
		TorrentFileToBuild.writeTempFile(TorrentFileToBuild.tempFileInit())
		TorrentFileToBuild.GetPeers()
		TorrentFileToBuild.pollGetPeersEveryCoupleMinutes()
		TorrentFileToBuild.downloadFileAsync()
		TorrentFileToBuild.writeFileInPieces("../output/", TorrentFileToBuild.Name, 0, TorrentFileToBuild.FileLength)
		TorrentFileToBuild.deleteTempFile()
	} else {
		totalSizeOfFile := getTotalSizeOfMulitpleFilesTorrent(&torrentInfo)
		TorrentFileToBuild := TorrentFileToBuild{}
		TorrentFileToBuild.LoadInfoHash(hash)
		TorrentFileToBuild.LoadPieceHashes(&torrentInfo)
		TorrentFileToBuild.LoadTrackers(&torrentInfo)
		TorrentFileToBuild.LoadMetaData(totalSizeOfFile, torrentInfo.Info.PieceLength)
		TorrentFileToBuild.writeTempFile(TorrentFileToBuild.tempFileInit())
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
		TorrentFileToBuild.deleteTempFile()
	}

}

func getHexHash(torrentPath string) (string, error) {
	cmd := exec.Command("python", "./CalculateHash.py", torrentPath)
	output, err := cmd.Output()
	if err != nil {
		log.Println(err)
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
