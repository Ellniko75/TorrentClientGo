package main

import (
	"encoding/binary"
	"fmt"
)

type TrackerResponse struct {
	ResponseAction      uint32
	ResponseTransaction uint32
	Interval            uint32
	Leechers            uint32
	Seeders             uint32
}

func (this *TrackerResponse) Create(trackerNetworkResponse []byte) {
	this.ResponseAction = binary.BigEndian.Uint32(trackerNetworkResponse[:4])
	this.ResponseTransaction = binary.BigEndian.Uint32(trackerNetworkResponse[4:8])
	this.Interval = binary.BigEndian.Uint32(trackerNetworkResponse[8:12])
	this.Leechers = binary.BigEndian.Uint32(trackerNetworkResponse[12:16])
	this.Seeders = binary.BigEndian.Uint32(trackerNetworkResponse[16:20])
}
func (this *TrackerResponse) Print() {
	printWithColor(Blue, "---------------------------------")
	printWithColor(Green, fmt.Sprint("Action> ", this.ResponseAction))
	printWithColor(Green, fmt.Sprint("transaction> ", this.ResponseTransaction))
	printWithColor(Green, fmt.Sprint("Interval> ", this.Interval))
	printWithColor(Green, fmt.Sprint("Leechers> ", this.Leechers))
	printWithColor(Green, fmt.Sprint("Seeders> ", this.Seeders))
	printWithColor(Blue, "---------------------------------")
}
