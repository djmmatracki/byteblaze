package main

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

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
	var hashes string
	var pieceLength int
	var piecesHaveEqualLength bool = true
	fileName := flag.String("file", "somefile", "a string")
	out := flag.String("out", ".torrent", "a string")
	flag.Parse()

	data, err := ioutil.ReadFile(*fileName)
	if err != nil {
		log.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		start, stop := i*len(data)/3, (i+1)*len(data)/3
		if i != 0 && len(data[start:stop]) != pieceLength {
			piecesHaveEqualLength = false
		}
		pieceLength = len(data[start:stop])
		log.Printf("piece length: %d\\n", pieceLength)
		log.Printf("piece: '%s'\n", data[start:stop])
		hash := sha1.Sum(data[start:stop])
		hashes += fmt.Sprintf("%s", hash)
	}
	file, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
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
	log.Printf("%x\n", hash)

	writer := bufio.NewWriter(file)
	err = bencode.Marshal(writer, tf)
	if err != nil {
		log.Fatal(err)
	}
	err = writer.Flush()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Pieces equal length:", piecesHaveEqualLength)
}
