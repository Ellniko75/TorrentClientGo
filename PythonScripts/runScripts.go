package pythonScripts

import (
	"encoding/hex"
	"log"
	"os/exec"
)

func GetInfoHash(torrentPath string) ([]byte, error) {
	cmd := exec.Command("python", "-u", "../pythonScripts/CalculateHash.py", torrentPath)
	output, err := cmd.Output()

	if err != nil {
		log.Println("first err", err)
		return nil, err
	}
	hexHash := string(output)
	hexHashWithoutShit := ""
	//remove all the shit that the output console has
	for _, v := range hexHash {
		if v == '\r' || v == '\n' {
			continue
		}
		hexHashWithoutShit += string(v)
	}
	//hexHashCorrectLength := hexHash[:len(hexHash)-2]

	hashToBytes, err := hex.DecodeString(hexHashWithoutShit)

	if err != nil {
		log.Println("error on GetInfoHash()", err)
		return nil, err
	}

	return hashToBytes, nil

}

func GetInfoHashNotShit(torrentPath string) {

}
