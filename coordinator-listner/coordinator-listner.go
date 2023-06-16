package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"net"

	"github.com/anacrolix/torrent/metainfo"
	"trrt-tst/byteblaze-deamon"
)

const (
	Destination = ""
)

func main() {
	conn, err := net.Dial("tcp", "143.42.54.122:5001")
	if err != nil {
		log.Fatalf("field to create conenction to %v", err)
	}
	defer conn.Close()

	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Read file data
	fileData, err := ioutil.ReadFile("default.conf")
	if err != nil {
		log.Fatal(err)
	}

	// Read torrent data
	torrentData, err := ioutil.ReadFile("torrentfile")
	if err != nil {
		log.Fatal(err)
	}
	mt, err := metainfo.Load(bytes.NewReader(torrentData))
	if err != nil {
		log.Fatal(err)
	}

	infoHash := fmt.Sprintf("%x", mt.HashInfoBytes())

	// Create and fill a Payload object
	payload := &byteblaze_deamon.Payload{
		File:         fileData,
		Torrent:      torrentData,
		FileName:     "default.conf",
		TorrentName:  "torrentfile",
		DropLocation: fmt.Sprintf("/var/byteblaze/%s", infoHash),
	}

	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err = encoder.Encode(payload)
	if err != nil {
		log.Fatal(err)
	}

	_, err = conn.Write(buffer.Bytes())
	if err != nil {
		log.Fatal(err)
	}
}
