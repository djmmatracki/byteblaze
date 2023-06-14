package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"

	"github.com/djmmatracki/byteblaze/internal/pkg/handshake"
	"github.com/jackpal/bencode-go"
)

type bencodeInfo struct {
	Pieces      string `bencode:"pieces"`
	PieceLength int    `bencode:"piece length"`
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
}

type TorrentFile struct {
	Announce string      `bencode:"announce"`
	Info     bencodeInfo `bencode:"info"`
}

func (i *bencodeInfo) hash() ([20]byte, error) {
	var buf bytes.Buffer
	err := bencode.Marshal(&buf, *i)
	if err != nil {
		return [20]byte{}, err
	}
	h := sha1.Sum(buf.Bytes())
	return h, nil
}

func main() {
	conn, err := net.Dial("tcp", "localhost:6881")
	if err != nil {
		log.Fatalln("error while dialing connection to localhost:6881")
		return
	}

	var hashes string
	var pieceLength int
	var piecesHaveEqualLength bool = true
	pieces := make([][]byte, 0)
	fileName := flag.String("file", "somefile", "a string")
	flag.Parse()

	// Readfile

	data, err := ioutil.ReadFile(*fileName)
	if err != nil {
		log.Fatal(err)
	}
	// Parse into pieces
	for i := 0; i < 3; i++ {
		start, stop := i*len(data)/3, (i+1)*len(data)/3 // TODO Make it not only 3
		if i != 0 && len(data[start:stop]) != pieceLength {
			piecesHaveEqualLength = false
		}
		pieceLength = len(data[start:stop])
		log.Printf("piece length: %d\\n", pieceLength)
		log.Printf("piece: '%s'\n", data[start:stop])
		pieces = append(pieces, data[start:stop])
		hash := sha1.Sum(data[start:stop])
		hashes += fmt.Sprintf("%s", hash)
	}

	tf := TorrentFile{
		Announce: "localhost:6337",
		Info: bencodeInfo{
			Length:      len(data),
			Pieces:      hashes,
			PieceLength: pieceLength,
			Name:        *fileName,
		},
	}
	hash, err := tf.Info.hash()
	if err != nil {
		return
	}
	log.Printf("infohash: %x\n", hash)

	// Save torrent file to /var/byteblaze/<infohash>
	torrentFilePath := fmt.Sprintf("/var/byteblaze/%x/torrentfile", hash)
	createDir(fmt.Sprintf("/var/byteblaze/%x", hash))
	file, err := os.Create(torrentFilePath)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer file.Close()

	err = bencode.Marshal(file, tf)
	if err != nil {
		return
	}

	log.Println("Pieces equal length:", piecesHaveEqualLength)
	// Save pieces
	for i, piece := range pieces {
		pieceFileName := fmt.Sprintf("/var/byteblaze/%x/%d", hash, i)
		f, err := os.Create(pieceFileName)
		if err != nil {
			return
		}
		f.Write(piece)
		f.Close()
	}

	// Send message to peer to broadcast torrentfile
	var peerID [20]byte
	_, err = rand.Read(peerID[:])
	if err != nil {
		return
	}

	req := handshake.New(handshake.HandshakeSendBroadcast, hash, peerID)
	_, err = conn.Write(req.Serialize())
	if err != nil {
		log.Println("error while sending handshake")
		return
	}

	_, err = handshake.Read(conn)
	if err != nil {
		log.Printf("cannot read handshake %v\n", err)
		return
	}

	err = bencode.Marshal(conn, tf)
	if err != nil {
		return
	}
}

func createDir(directoryPath string) {
	// Check if the directory already exists
	if _, err := os.Stat(directoryPath); os.IsNotExist(err) {
		// Directory does not exist, create it
		err := os.MkdirAll(directoryPath, 0755)
		if err != nil {
			fmt.Println("Failed to create directory:", err)
			return
		}
		fmt.Println("Directory created:", directoryPath)
	} else {
		fmt.Println("Directory already exists:", directoryPath)
	}
}
