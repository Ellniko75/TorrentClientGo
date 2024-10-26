package main

import (
	"crypto/sha1"
	"fmt"
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

func WriteToOkstxt() {
	file, err := os.OpenFile("Oks.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close() // Ensure the file is closed after we're done

	_, err = file.WriteString(" OK \n")

	if err != nil {
		fmt.Println("Error writing to file")
	}
}

func WriteToErrorstxt() {
	file, err := os.OpenFile("Errors.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close() // Ensure the file is closed after we're done

	_, err = file.WriteString(" BAD \n")

	if err != nil {
		fmt.Println("Error writing to file")
	}
}
