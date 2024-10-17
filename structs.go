package main

type TorrentFile struct {
	Announce string `bencode:"announce"`
	Info     struct {
		Pieces      string `bencode:"pieces"`
		PieceLength int    `bencode:"piece length"`
	} `bencode:"info"`
	AnnounceList [][]string `bencode:"announce-list"` // Optional multiple trackers
}

type TorrentFileToBuild struct {
	ListOfTrackers []string   //list of all the trackers
	ListOfHashes   [][20]byte //hashes for each piece of the file
	File           []byte     //property to write the file when the pieces arrive
}

type Error struct {
	FunctionName string
	ErrorName    string
}
