package main

import (
	"bytes"
	"encoding/gob"
	"io/ioutil"
	"log"
	"net"

	"trrt-tst/byteblaze-deamon"
)

const (
	Destination = ""
)

func main() {
	conn, err := net.Dial("tcp", "localhost:5001")
	defer conn.Close()

	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Read file data
	fileData, err := ioutil.ReadFile("/root/dummyfile")
	if err != nil {
		log.Fatal(err)
	}

	// Read torrent data
	torrentData, err := ioutil.ReadFile("/root/test.torrent")
	if err != nil {
		log.Fatal(err)
	}

	// Create and fill a Payload object
	payload := &byteblaze_deamon.Payload{
		File:         fileData,
		Torrent:      torrentData,
		FileName:     "myfile",
		TorrentName:  "myfile.torrent",
		DropLocation: "/root/drop",
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
