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

func GetExpectedFile() []byte {
	file, err := os.Open("./xokas.mp4")
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil
	}

	fileRead := make([]byte, fileInfo.Size())
	_, err = file.Read(fileRead)
	if err != nil {
		return nil
	}

	return fileRead
}
