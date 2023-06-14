// package main

// import (
// 	"crypto/sha1"
// 	"io/ioutil"
// 	"log"
// 	"os"

// 	bencode "github.com/jackpal/bencode-go"
// )

// type TorrentFile struct {
// 	Announce string `bencode:"announce"`
// 	Info     Info   `bencode:"info"`
// }

// type Info struct {
// 	PieceLength int64  `bencode:"piece length"`
// 	Pieces      string `bencode:"pieces"`
// 	Length      int64  `bencode:"length"`
// 	Name        string `bencode:"name"`
// }

// func main() {
// 	data, err := ioutil.ReadFile("dummyfile")
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	pieceLength := int64(256 * 1024) // typically piece sizes are powers of two, like 256KB
// 	numPieces := len(data) / int(pieceLength)
// 	if len(data)%int(pieceLength) != 0 {
// 		numPieces++
// 	}

// 	pieces := ""
// 	for i := 0; i < numPieces; i++ {
// 		start := i * int(pieceLength)
// 		end := start + int(pieceLength)
// 		if end > len(data) {
// 			end = len(data)
// 		}
// 		hash := sha1.Sum(data[start:end])
// 		pieces += string(hash[:])
// 	}

// 	info := Info{
// 		PieceLength: pieceLength,
// 		Pieces:      pieces,
// 		Length:      int64(len(data)),
// 		Name:        "dummyfile",
// 	}

// 	torrent := TorrentFile{
// 		Announce: "http://my.tracker/announce",
// 		Info:     info,
// 	}

// 	file, err := os.Create("my.torrent")
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer file.Close()

// 	err = bencode.Marshal(file, torrent)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// }
