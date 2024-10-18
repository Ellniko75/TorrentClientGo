package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"time"
)

func createTcpConnection(url string) (net.Conn, error) {
	// Connect to the server
	fmt.Println(url)
	conn, err := net.DialTimeout("tcp", url, 10*time.Second)
	if err != nil {
		return nil, createError("createTcpConnection()", err.Error())
	}

	return conn, nil
}

func connectToPeerAndRequestFile(ip string, fileIndex int, fileLength int) ([]byte, error) {
	//create the tcp connection
	conn, err := createTcpConnection(ip)
	if err != nil {
		return nil, err
	}

	//load the payload to send to the peer
	var buff bytes.Buffer

	//requestSize, always 13
	if err = binary.Write(&buff, binary.BigEndian, int32(13)); err != nil {
		log.Println(err)
	}
	//indicate that this is a request with the 6
	if err = binary.Write(&buff, binary.BigEndian, byte(6)); err != nil {
		log.Println(err)
	}
	//file index
	if err = binary.Write(&buff, binary.BigEndian, int32(fileIndex)); err != nil {
		log.Println(err)
	}
	//begin index
	if err = binary.Write(&buff, binary.BigEndian, int32(0)); err != nil {
		log.Println(err)
	}
	//file size
	if err = binary.Write(&buff, binary.BigEndian, int32(fileLength)); err != nil {
		log.Println(err)
	}

	//send the payload requesting the file
	n, err := conn.Write(buff.Bytes())
	if n == 0 || err != nil {
		return nil, createError("connectToPeerAndRequestFile() on conn.Write()", err.Error())
	}

	//read the data
	response := []byte{}
	n, err = conn.Read(response)
	if n == 0 || err != nil {
		return nil, createError("connectToPeerAndRequestFile() on conn.Read()", err.Error())
	}

	return response, nil
}
