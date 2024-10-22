package main

import (
	"encoding/binary"
	"fmt"
)

// Contains all the information of the response of the tracker, including number of seeders, and the IP addresses
type TrackerResponse struct {
	ResponseAction         uint32 //type of response 3->error
	ResponseTransaction    uint32 //transaction id
	Interval               uint32 //interval in ms
	Leechers               uint32 //number of leechers
	Seeders                uint32 //number of seeders
	TrackerNetworkResponse []byte //Slice of ips and ports in pure bytes
}

// Creates the Tracker Reponse
func (this *TrackerResponse) Create(trackerNetworkResponse []byte) {
	this.ResponseAction = binary.BigEndian.Uint32(trackerNetworkResponse[:4])
	this.ResponseTransaction = binary.BigEndian.Uint32(trackerNetworkResponse[4:8])
	this.Interval = binary.BigEndian.Uint32(trackerNetworkResponse[8:12])
	this.Leechers = binary.BigEndian.Uint32(trackerNetworkResponse[12:16])
	this.Seeders = binary.BigEndian.Uint32(trackerNetworkResponse[16:20])
	this.TrackerNetworkResponse = trackerNetworkResponse[20:]
}

// shows all the metainfo about the response
func (this *TrackerResponse) Print() {
	printWithColor(Blue, "---------------------------------")
	printWithColor(Green, fmt.Sprint("Action> ", this.ResponseAction))
	printWithColor(Green, fmt.Sprint("transaction> ", this.ResponseTransaction))
	printWithColor(Green, fmt.Sprint("Interval> ", this.Interval))
	printWithColor(Green, fmt.Sprint("Leechers> ", this.Leechers))
	printWithColor(Green, fmt.Sprint("Seeders> ", this.Seeders))
	printWithColor(Blue, "---------------------------------")
}

// returns a []string of all the ips and ports already processed, ex: 195.234.55.1:8080
func (this *TrackerResponse) getIpAndPorts() []string {
	ipsParsed := []string{}

	for i := 0; i < int(this.Seeders)*6; i = i + 6 {
		first := this.TrackerNetworkResponse[i]
		second := this.TrackerNetworkResponse[i+1]
		third := this.TrackerNetworkResponse[i+2]
		fourth := this.TrackerNetworkResponse[i+3]
		port := this.TrackerNetworkResponse[i+4 : i+6]
		portDecimal := binary.BigEndian.Uint16(port)
		//TODO: FIX THE WAY IT INTERPRETS THE PORTS SINCE ITS WRONG - aparently its not wrong lmao
		//Get complete IP
		ip := fmt.Sprint(first, ".", second, ".", third, ".", fourth, ":", portDecimal)

		//This will mean that we are at the point of the slice where all the rest of the ips are zeroes.
		//It happens because GO needs a slice with a fixed size to get a network response, so you have to initialize it with all zeroes
		//and if not enough IPS are given, then the rest of the space will be zeroes
		if ip == "0.0.0.0:0" {
			break
		}
		ipsParsed = append(ipsParsed, ip)
	}
	return ipsParsed
}
