package main

import (
	"encoding/binary"
	"fmt"
)

type TrackerResponse struct {
	ResponseAction      uint32 //type of response 3->error
	ResponseTransaction uint32 //transaction id
	Interval            uint32 //interval in ms
	Leechers            uint32 //number of leechers
	Seeders             uint32 //number of seeders
	IpsAndPorts         []byte //Slice of ips and ports in pure bytes
}

// creates the trackerResponse
func (this *TrackerResponse) Create(trackerNetworkResponse []byte) {
	this.ResponseAction = binary.BigEndian.Uint32(trackerNetworkResponse[:4])
	this.ResponseTransaction = binary.BigEndian.Uint32(trackerNetworkResponse[4:8])
	this.Interval = binary.BigEndian.Uint32(trackerNetworkResponse[8:12])
	this.Leechers = binary.BigEndian.Uint32(trackerNetworkResponse[12:16])
	this.Seeders = binary.BigEndian.Uint32(trackerNetworkResponse[16:20])
	this.IpsAndPorts = trackerNetworkResponse[20:]

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
		first := this.IpsAndPorts[i]
		second := this.IpsAndPorts[i+1]
		third := this.IpsAndPorts[i+2]
		fourth := this.IpsAndPorts[i+3]
		port := this.IpsAndPorts[i+4 : i+6]
		portDecimal := binary.BigEndian.Uint16(port)
		//TODO: FIX THE WAY IT INTERPRETS THE PORTS SINCE ITS WRONG - aparently its not wrong lmao
		//Get complete IP
		ip := fmt.Sprint(first, ".", second, ".", third, ".", fourth, ":", portDecimal)
		ipsParsed = append(ipsParsed, ip)
	}
	return ipsParsed
}
