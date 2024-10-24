package main

import (
	"crypto/sha1"
	"os"
)

func GetSha1Hash(toHash []byte) []byte {
	hash := sha1.New()
	hash.Write(toHash)
	hashBytes := hash.Sum(nil)

	return hashBytes
}

func GetExpectedBytes(indexStart int, indexFinish int) ([]byte, error) {
	file, err := os.Open("./xokas.mp4")
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	fileRead := make([]byte, fileInfo.Size())
	_, err = file.Read(fileRead)
	if err != nil {
		return nil, err
	}

	return fileRead[indexStart:indexFinish], nil

}
