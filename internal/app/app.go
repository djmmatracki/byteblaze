package app

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/djmmatracki/byteblaze/internal/pkg/bitfield"
	"github.com/djmmatracki/byteblaze/internal/pkg/handshake"
	"github.com/djmmatracki/byteblaze/internal/pkg/message"
	"github.com/djmmatracki/byteblaze/internal/pkg/torrentfile"
	"github.com/jackpal/bencode-go"
)

func Run() {
	listen, err := net.Listen("tcp", "0.0.0.0:6881")
	if err != nil {
		log.Fatal(err)
	}
	defer listen.Close()

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("received connection")
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	// Complete handshake
	log.Printf("reading handshake from %s to %s\n", conn.RemoteAddr(), conn.LocalAddr())
	hs, err := handshake.Read(conn)
	if err != nil {
		log.Printf("cannot read handshake %v\n", err)
		return
	}
	log.Printf("got action %d\n", hs.Action)

	// Read actions
	switch hs.Action {
	case handshake.HandshakeReceiveBroadcast:
		// Receive a broadcast message with the torrentfile and start downloading the file specified in the torrent file.
		log.Println("received broadcast and start downloading")
		// Send back handshake
		req := handshake.New(handshake.HandshakeACK, hs.InfoHash, hs.PeerID)
		_, err := conn.Write(req.Serialize())
		if err != nil {
			log.Println("error while sending handshake")
			return
		}

		log.Println("sent ACK to host")
		// Read torrentfile
		bto := torrentfile.BencodeTorrent{}

		// TODO - Save torrentfile to file system /var/byteblaze
		log.Println("starting to read...")
		data, err := io.ReadAll(conn)
		if err != nil {
			log.Fatal("error while reading torrentfile data")
			return
		}
		log.Println("creating directory")
		createDir(fmt.Sprintf("/var/byteblaze/%x", hs.InfoHash))
		torrentFilePath := fmt.Sprintf("/var/byteblaze/%x/torrentfile", hs.InfoHash)
		err = os.WriteFile(torrentFilePath, data, 0666) // Make specific file perms
		if err != nil {
			log.Fatal("error while writing torrentfile")
			return
		}
		log.Println("created torrentfile")

		log.Printf("unmarshaling to bencode format, data: '%s'", data)
		buffer := bytes.NewBuffer(data)
		err = bencode.Unmarshal(buffer, &bto)
		if err != nil {
			log.Fatal("cannot unmarshal beencode file")
			return
		}
		log.Printf("unmarshaled beencode file to %+v", bto)

		tf, err := bto.ToTorrentFile()
		if err != nil {
			log.Fatal("cannot convert to TorrentFile")
			return
		}
		log.Println("converted bencode to torrentfile")

		// Start downloading
		log.Println("starting downloading")
		err = tf.DownloadToFile("downloaded_file")
		if err != nil {
			log.Fatal(err)
		}

	case handshake.HandshakeRequest:
		// Request for a given file. When some other peer trys to download a file.
		//   - Based on the info hash identify what file does the client want
		//   - Create a map that will based on the infohash get the directory of the pieces

		// Return back the handshake
		// TODO Change this to actual info hash
		log.Printf("received request for a file with infohash %s", hs.InfoHash)
		req := handshake.New(handshake.HandshakeACK, hs.InfoHash, hs.PeerID)

		_, err := conn.Write(req.Serialize())
		if err != nil {
			log.Println("error while sending handshake")
			return
		}
		log.Println("handshake completed succesfully")
		err = handleFileRequest(conn, hs.InfoHash)
		if err != nil {
			log.Printf("error while preocessing request: %v", err)
			return
		}
		log.Println("completed request")
	case handshake.HandshakeSendBroadcast:
		// Receive a message with the torrentfile, broadcast it to other peers and start download the file specified in the torrent file.
		// Send back handshake with ACK
		// Expect torrentfile
		// Send torrentfile to peers
		// Start downloading
		log.Println("received send broadcast and start downloading")
		// Send back handshake
		req := handshake.New(handshake.HandshakeACK, hs.InfoHash, hs.PeerID)
		_, err := conn.Write(req.Serialize())
		if err != nil {
			log.Println("error while sending handshake")
			return
		}

		log.Println("sent ACK to host")
		// Read torrentfile
		bto := torrentfile.BencodeTorrent{}

		// TODO - Save torrentfile to file system /var/byteblaze

		err = bencode.Unmarshal(conn, &bto)
		if err != nil {
			log.Fatal("cannot unmarshal beencode file")
			return
		}
		log.Println("unmarshaling beencode file")

		tf, err := bto.ToTorrentFile()
		if err != nil {
			log.Fatal("cannot convert to TorrentFile")
			return
		}
		log.Println("converted bencode to torrentfile")
		log.Println("broadcasting messages to peers")
		// Send broadcast to peers
		peers := []string{
			"143.42.54.125:6881",
			"143.42.54.140:6881",
		}
		for _, peerIP := range peers {
			broadcastMessage(peerIP, bto, hs.InfoHash)
		}
		log.Println("broadcasted messages")

		// Start downloading
		log.Println("starting downloading")
		err = tf.DownloadToFile("downloadedfile")
		if err != nil {
			log.Fatal(err)
		}
	default:
		log.Println("corrupted handshake")
	}

}

func handleFileRequest(conn net.Conn, infoHash [20]byte) error {
	// Read files from /var/byteblaze
	// Check what info hashes are in the folder
	// Check what pieces do I have
	// Compose bitfield
	bf := make(bitfield.Bitfield, 1)
	pieces := make(map[int][]byte)

	log.Printf("got info hash %x", infoHash)
	pathToPieces := fmt.Sprintf("/var/byteblaze/%x", infoHash)

	_, err := os.Stat(pathToPieces)
	if err == nil {
		log.Println("File exists")
	} else if os.IsNotExist(err) {
		log.Println("File does not exist")
		return err
	} else {
		log.Println("error")
		return err
	}

	log.Println("reading directory pieces")
	dir, err := os.ReadDir(pathToPieces)
	if err != nil {
		log.Printf("error while reading dir %s", pathToPieces)
		return err
	}

	for _, file := range dir {
		if file.Name() == "torrentfile" {
			continue
		}
		log.Printf("processing piece '%s'", file.Name())
		i, err := strconv.Atoi(file.Name())
		if err != nil {
			log.Printf("error while converting file name to int: %v", err)
			return err
		}
		filePath := fmt.Sprintf("%s/%s", pathToPieces, file.Name())
		log.Printf("processing file path %s", filePath)
		f, err := os.Open(filePath)
		if err != nil {
			return err
		}
		log.Println("sucessfuly opened piece file")
		pieces[i], err = ioutil.ReadAll(f)
		if err != nil {
			log.Println("error while reading piece")
			return err
		}
		log.Println("succesfuly read piece")
		f.Close()
		bf.SetPiece(i)
		log.Printf("set bitfield %x", bf)
	}

	// Send bitfield
	bitFieldMsg := message.Message{
		ID:      message.MsgBitfield,
		Payload: bf,
	}
	log.Printf("sending bitfield %x", bf)
	_, err = conn.Write(bitFieldMsg.Serialize())
	if err != nil {
		log.Println("unable to send message")
		return err
	}
	log.Println("sending messages")

	for {
		// Listen for messages
		msg, err := message.Read(conn)
		if err != nil {
			return err
		}

		if msg == nil {
			return err
		}

		switch msg.ID {
		case message.MsgUnchoke:
			log.Println("got unchoke message")
			msg := message.Message{
				ID: message.MsgUnchoke,
			}
			conn.Write(msg.Serialize())
		case message.MsgChoke:
			log.Println("got choke message")
			msg := message.Message{
				ID: message.MsgChoke,
			}
			conn.Write(msg.Serialize())
		case message.MsgInterested:
			log.Println("got interested message")
		case message.MsgRequest:
			log.Println("got request message")
			if len(msg.Payload) != 12 {
				log.Println("got invalid length of payload for message request")
				return err
			}
			index := int(binary.BigEndian.Uint32(msg.Payload[0:4]))
			begin := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
			length := int(binary.BigEndian.Uint32(msg.Payload[8:12]))
			payload := make([]byte, 8+length)
			copy(payload[0:8], msg.Payload[0:8])
			copy(payload[8:], pieces[index][begin:])

			msgWithPiece := message.Message{
				ID:      message.MsgPiece,
				Payload: payload,
			}
			// Send back the message with the piece
			conn.Write(msgWithPiece.Serialize())
		case message.MsgHave:
			log.Println("Got message have")
		default:
			log.Println("Undefined message")
			return err
		}

	}
}

func broadcastMessage(peerIP string, tf torrentfile.BencodeTorrent, infoHash [20]byte) {
	conn, err := net.Dial("tcp", peerIP)
	if err != nil {
		return
	}
	var peerID [20]byte
	_, err = rand.Read(peerID[:])
	if err != nil {
		return
	}

	req := handshake.New(handshake.HandshakeReceiveBroadcast, infoHash, peerID)
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
		log.Printf("cannot marshal tf to connection %v\n", err)
		return
	}
	defer conn.Close()
	log.Printf("successfuly broadcasted message to peer %s", peerIP)
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
