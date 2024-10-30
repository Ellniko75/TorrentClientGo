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

func WriteToOkstxt(fileIndex int) {
	file, err := os.OpenFile("Oks.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close() // Ensure the file is closed after we're done

	_, err = file.WriteString(fmt.Sprint("Succesfully downloaded Piece: ", fileIndex, "\n"))

	if err != nil {
		fmt.Println("Error writing to file")
	}
}

func WriteToErrorstxt(fileIndex int) {
	file, err := os.OpenFile("Errors.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close() // Ensure the file is closed after we're done

	_, err = file.WriteString(fmt.Sprint("Error on index: ", fileIndex, "\n"))

	if err != nil {
		fmt.Println("Error writing to file")
	}
}

func ResetOksAndErrors() {
	err := os.Truncate("Errors.txt", 0)
	if err != nil {

		fmt.Println("Errors.txt cannot be cleaned because they don't Exist")
	}

	err = os.Truncate("Oks.txt", 0)
	if err != nil {
		fmt.Println("Oks.txt cannot be cleaned because they don't Exist")
	}
}

func CheckPlacesWhereTheBytesAreDifferent(reference []byte, downloaded []byte) int {
	for i, v := range downloaded {
		if v != reference[i] {
			return i
		}
	}
	return 0
}
func contains(value string, slice []string) bool {

	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

func getDivisibleNumber(number int) int {
	for i := 3; i < 10000; i++ {
		if number%i == 0 {
			return i
		}
	}
	return 0
}
